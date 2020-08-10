package controller

import (
	"context"
	"testing"
	"time"

	"github.com/fortytw2/leaktest"
	"github.com/google/go-cmp/cmp"
	routev1 "github.com/openshift/api/route/v1"
	fakeproject "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1/fake"
	fakerouterclient "github.com/openshift/client-go/route/clientset/versioned/fake"
	kapi "k8s.io/api/core/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	fakekubeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openshift/router/pkg/router"
	"github.com/openshift/router/pkg/router/controller/endpointsubset"
)

const endpointSliceTestTimeout = 1 * time.Minute

type noopTestPlugin struct{}

// Ensure noopTestPlugin is a router.Plugin.
var _ router.Plugin = (*noopTestPlugin)(nil)

func (p *noopTestPlugin) HandleRoute(watch.EventType, *routev1.Route) error {
	return nil
}

func (p *noopTestPlugin) HandleNamespaces(sets.String) error {
	return nil
}

func (p *noopTestPlugin) HandleNode(watch.EventType, *kapi.Node) error {
	return nil
}

func (p *noopTestPlugin) HandleEndpoints(watch.EventType, *kapi.Endpoints) error {
	return nil
}

func (p *noopTestPlugin) Commit() error {
	return nil
}

type handleEndpointsEvent struct {
	eventType watch.EventType
	endpoints *kapi.Endpoints
}

type endpointSlicesTestPlugin struct {
	noopTestPlugin

	handleEndpointsCh chan handleEndpointsEvent
}

// Ensure endpointSlicesTestPlugin is a router.Plugin.
var _ router.Plugin = (*endpointSlicesTestPlugin)(nil)

func (p endpointSlicesTestPlugin) HandleEndpoints(eventType watch.EventType, endpoints *kapi.Endpoints) error {
	p.handleEndpointsCh <- handleEndpointsEvent{
		eventType: eventType,
		endpoints: endpoints,
	}
	return nil
}

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}

// stringPtr returns a pointer to the passed string.
func stringPtr(s string) *string {
	return &s
}

// protocolPtr returns a pointer to the passed protocol.
func protocolPtr(p kapi.Protocol) *kapi.Protocol {
	return &p
}

func newEndpointSliceTestController(plugin router.Plugin, initialObjects ...runtime.Object) (*fakekubeclient.Clientset, *RouterController, chan struct{}) {
	stopCh := make(chan struct{})
	client := fakekubeclient.NewSimpleClientset(initialObjects...)

	factory := NewDefaultRouterControllerFactory(
		fakerouterclient.NewSimpleClientset(),
		&fakeproject.FakeProjects{},
		client,
		false, // watch endpoints
	)

	controller := &RouterController{
		Plugin:             plugin,
		NamespaceEndpoints: make(map[string]map[string]*kapi.Endpoints),
	}

	// The order here is signficant. In factory.Create() we
	// register the event handlers after starting the informers
	// but I find that to deliver (racy?) inconsistent callbacks
	// on the plugin when the event handlers are subsequently
	// added after both the informers are running and populating
	// the initial store. Here we don't care about the initial
	// sync, so skip that part.
	factory.initInformers(controller)
	factory.registerInformerEventHandlers(controller)
	factory.startInformers(stopCh)

	return client, controller, stopCh
}

func TestEndpointSlicesAdd(t *testing.T) {
	defer leaktest.CheckTimeout(t, endpointSliceTestTimeout)()

	plugin := &endpointSlicesTestPlugin{
		handleEndpointsCh: make(chan handleEndpointsEvent),
	}

	client, controller, stopCh := newEndpointSliceTestController(plugin)
	defer close(stopCh)
	controller.Run()

	if !controller.firstSyncDone {
		t.Fatalf("expected first sync to be completed")
	}

	type testCase struct {
		sliceToAdd              discoveryv1beta1.EndpointSlice
		expectedServiceName     string
		expectedEventType       watch.EventType
		expectedEndpointSubsets []kapi.EndpointSubset
	}

	testCases := []testCase{{
		sliceToAdd: discoveryv1beta1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "slice-1",
				Namespace: "namespace-a",
				Labels: map[string]string{
					ServiceNameLabel: "service-a",
				},
			},
			AddressType: discoveryv1beta1.AddressTypeIPv4,
			Endpoints: []discoveryv1beta1.Endpoint{{
				Addresses: []string{
					"192.168.0.1",
					"10.0.0.1",
					"172.17.10.8",
				},
				Hostname: stringPtr("service.com"),
			}},
			Ports: []discoveryv1beta1.EndpointPort{{
				Port: int32Ptr(8080),
			}, {
				Name:     stringPtr("https"),
				Protocol: protocolPtr(kapi.ProtocolTCP),
				Port:     int32Ptr(443),
			}, {
				Name:     stringPtr("http"),
				Protocol: protocolPtr(kapi.ProtocolTCP),
				Port:     int32Ptr(80),
			}},
		},
		expectedServiceName: "service-a",
		expectedEventType:   watch.Modified,
		expectedEndpointSubsets: []kapi.EndpointSubset{{
			Addresses: []kapi.EndpointAddress{{
				IP:       "10.0.0.1",
				Hostname: "service.com",
			}, {
				IP:       "172.17.10.8",
				Hostname: "service.com",
			}, {
				IP:       "192.168.0.1",
				Hostname: "service.com",
			}},
			Ports: []kapi.EndpointPort{{
				Name:     "http",
				Port:     80,
				Protocol: "TCP",
			}, {
				Name:     "https",
				Port:     443,
				Protocol: "TCP",
			}, {
				Port: 8080,
			}},
		}},
	}, {
		sliceToAdd: discoveryv1beta1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "slice-2",
				Namespace: "namespace-a",
				Labels: map[string]string{
					ServiceNameLabel: "service-a",
				},
			},
			AddressType: discoveryv1beta1.AddressTypeIPv4,
			Endpoints: []discoveryv1beta1.Endpoint{{
				Addresses: []string{
					"172.16.10.2",
					"172.16.10.1",
				},
			}},
			Ports: []discoveryv1beta1.EndpointPort{{
				Port: int32Ptr(101),
			}, {
				Port: int32Ptr(100),
			}},
		},
		expectedServiceName: "service-a",
		expectedEventType:   watch.Modified,
		expectedEndpointSubsets: []kapi.EndpointSubset{{
			Addresses: []kapi.EndpointAddress{{
				IP:       "10.0.0.1",
				Hostname: "service.com",
			}, {
				IP:       "172.17.10.8",
				Hostname: "service.com",
			}, {
				IP:       "192.168.0.1",
				Hostname: "service.com",
			}},
			Ports: []kapi.EndpointPort{{
				Name:     "http",
				Port:     80,
				Protocol: "TCP",
			}, {
				Name:     "https",
				Port:     443,
				Protocol: "TCP",
			}, {
				Port: 8080,
			}},
		}, {
			Addresses: []kapi.EndpointAddress{{
				IP: "172.16.10.1",
			}, {
				IP: "172.16.10.2",
			}},
			Ports: []kapi.EndpointPort{{
				Port: 100,
			}, {
				Port: 101,
			}},
		}},
	}, {
		sliceToAdd: discoveryv1beta1.EndpointSlice{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "slice-1",
				Namespace: "namespace-b",
				Labels: map[string]string{
					ServiceNameLabel: "service-b",
				},
			},
		},
		expectedServiceName:     "service-b",
		expectedEventType:       watch.Modified,
		expectedEndpointSubsets: []kapi.EndpointSubset{{}},
	}}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if _, err := client.DiscoveryV1beta1().EndpointSlices(tc.sliceToAdd.Namespace).Create(context.TODO(), &tc.sliceToAdd, metav1.CreateOptions{}); err != nil {
				t.Fatalf("failed to create endpointslice %s: %v", tc.sliceToAdd.Name, err)
			}

			var event handleEndpointsEvent

			select {
			case event = <-plugin.handleEndpointsCh:
			case <-time.After(endpointSliceTestTimeout):
				t.Fatal("timeout")
			}

			if event.eventType != tc.expectedEventType {
				t.Errorf("expected event type %q, got %q", tc.expectedEventType, event.eventType)
			}

			if event.endpoints.Name != tc.expectedServiceName {
				t.Errorf("expected service %q, got %q", tc.expectedServiceName, event.endpoints.Name)
			}

			if diff := cmp.Diff(tc.expectedEndpointSubsets, event.endpoints.Subsets); len(diff) > 0 {
				t.Errorf("mismatched endpoint subsets (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEndpointSlicesDelete(t *testing.T) {
	defer leaktest.CheckTimeout(t, endpointSliceTestTimeout)()

	serviceName := "service-a"

	eps1 := discoveryv1beta1.EndpointSlice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "endpointslices",
			APIVersion: "discovery.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slice-1",
			Namespace: "namespace-a",
			Labels: map[string]string{
				ServiceNameLabel: serviceName,
			},
		},
		AddressType: discoveryv1beta1.AddressTypeIPv4,
		Endpoints: []discoveryv1beta1.Endpoint{{
			Addresses: []string{
				"192.168.0.1",
				"10.0.0.1",
				"172.17.10.8",
			},
			Hostname: stringPtr("service.com"),
		}},
		Ports: []discoveryv1beta1.EndpointPort{{
			Port: int32Ptr(8080),
		}, {
			Name:     stringPtr("https"),
			Protocol: protocolPtr(kapi.ProtocolTCP),
			Port:     int32Ptr(443),
		}, {
			Name:     stringPtr("http"),
			Protocol: protocolPtr(kapi.ProtocolTCP),
			Port:     int32Ptr(80),
		}},
	}

	eps2 := discoveryv1beta1.EndpointSlice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "endpointslices",
			APIVersion: "discovery.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slice-2",
			Namespace: "namespace-a",
			Labels: map[string]string{
				ServiceNameLabel: serviceName,
			},
		},
		AddressType: discoveryv1beta1.AddressTypeIPv4,
		Endpoints: []discoveryv1beta1.Endpoint{{
			Addresses: []string{
				"172.16.10.2",
				"172.16.10.1",
			},
		}},
		Ports: []discoveryv1beta1.EndpointPort{{
			Port: int32Ptr(101),
		}, {
			Port: int32Ptr(100),
		}},
	}

	initialObjs := []runtime.Object{&eps1, &eps2}
	client, controller, stopCh := newEndpointSliceTestController(&noopTestPlugin{}, initialObjs...)
	defer close(stopCh)
	controller.Run()

	if err := wait.PollImmediate(100*time.Millisecond, endpointSliceTestTimeout, func() (done bool, err error) {
		objs, err := client.DiscoveryV1beta1().EndpointSlices(eps1.Namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		if !controller.firstSyncDone {
			return false, nil
		}
		s1 := convertEndpointSliceToEndpointSubset(objs.Items,
			endpointsubset.DefaultEndpointAddressOrderByFuncs(),
			endpointsubset.DefaultEndpointPortOrderByFuncs())

		s2 := convertEndpointSliceToEndpointSubset([]discoveryv1beta1.EndpointSlice{eps1, eps2},
			endpointsubset.DefaultEndpointAddressOrderByFuncs(),
			endpointsubset.DefaultEndpointPortOrderByFuncs())

		return len(cmp.Diff(s1, s2)) == 0, nil
	}); err != nil {
		t.Fatalf("initial setup failed: %v", err)
	}

	// Now that we've reached our initial state we want to receive
	// events when HandleEndpoints().
	plugin := &endpointSlicesTestPlugin{
		handleEndpointsCh: make(chan handleEndpointsEvent),
	}

	// Prevent data race when replacing plugin.
	controller.lock.Lock()
	controller.Plugin = plugin
	controller.lock.Unlock()

	type testCase struct {
		description             string
		sliceToDelete           discoveryv1beta1.EndpointSlice
		expectedEndpointSubsets []kapi.EndpointSubset
		expectedEventType       watch.EventType
		expectDeleteError       bool
	}

	testCases := []testCase{{
		sliceToDelete: eps1,
		expectedEndpointSubsets: []kapi.EndpointSubset{{
			Addresses: []kapi.EndpointAddress{{
				IP: "172.16.10.1",
			}, {
				IP: "172.16.10.2",
			}},
			Ports: []kapi.EndpointPort{{
				Port: 100,
			}, {
				Port: 101,
			}},
		}},
		expectedEventType: watch.Modified,
	}, {
		sliceToDelete:     eps2,
		expectedEventType: watch.Deleted,
	}, {
		sliceToDelete:     eps1,
		expectDeleteError: true,
	}, {
		sliceToDelete:     eps2,
		expectDeleteError: true,
	}}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			if err := client.DiscoveryV1beta1().EndpointSlices(tc.sliceToDelete.Namespace).Delete(context.TODO(), tc.sliceToDelete.Name, metav1.DeleteOptions{}); err != nil {
				if tc.expectDeleteError {
					return
				}
				t.Fatalf("failed to delete endpointslice %s: %v", eps1.Name, err)
			}

			var event handleEndpointsEvent

			select {
			case event = <-plugin.handleEndpointsCh:
			case <-time.After(endpointSliceTestTimeout):
				t.Fatal("timeout")
			}

			if actual := event.eventType; actual != tc.expectedEventType {
				t.Fatalf("expected event type %q, got %q", tc.expectedEventType, actual)
			}

			if diff := cmp.Diff(tc.expectedEndpointSubsets, event.endpoints.Subsets); len(diff) > 0 {
				t.Fatalf("mismatched endpoint subsets (-want +got):\n%s", diff)
			}
		})
	}
}
