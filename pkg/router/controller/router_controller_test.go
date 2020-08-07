package controller

import (
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"sync/atomic"
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
	"k8s.io/apimachinery/pkg/watch"
	fakekubeclient "k8s.io/client-go/kubernetes/fake"

	"github.com/openshift/router/pkg/router"
	"github.com/openshift/router/pkg/router/controller/endpointsubset"
)

const endpointSliceTestTimeout = 1 * time.Minute

type endpointSlicesTestPlugin struct {
	commitCalled  chan struct{}
	commitCount   uint64
	lastEventType watch.EventType
	endpoints     map[string]kapi.Endpoints
}

// Ensure endpointSlicesTestPlugin is a router.Plugin.
var _ router.Plugin = (*endpointSlicesTestPlugin)(nil)

func (p *endpointSlicesTestPlugin) HandleRoute(watch.EventType, *routev1.Route) error {
	panic("should not be called")
}

func (p *endpointSlicesTestPlugin) HandleNamespaces(sets.String) error {
	panic("should not be called")
}

func (p *endpointSlicesTestPlugin) HandleNode(watch.EventType, *kapi.Node) error {
	panic("should not be called")
}

func (p *endpointSlicesTestPlugin) HandleEndpoints(eventType watch.EventType, endpoints *kapi.Endpoints) error {
	if eventType == watch.Deleted {
		delete(p.endpoints, path.Join(endpoints.Namespace, endpoints.Name))
	} else {
		p.endpoints[path.Join(endpoints.Namespace, endpoints.Name)] = *endpoints
	}

	p.lastEventType = eventType
	//	fmt.Printf("****************************** PLUGIN HandleEndpoints: %q, %+v\n", eventType, endpoints)
	return nil
}

func (p *endpointSlicesTestPlugin) Commit() error {
	atomic.AddUint64(&p.commitCount, 1)
	p.commitCalled <- struct{}{}
	return nil
}

func (p *endpointSlicesTestPlugin) GetCommitCount() uint64 {
	return atomic.LoadUint64(&p.commitCount)
}

func (p *endpointSlicesTestPlugin) WaitForNCommits(n int, timeout time.Duration) error {
	for i := 0; i < n; i++ {
		select {
		case <-p.commitCalled:
		case <-time.After(timeout):
			return errors.New("commit timeout")
		}
	}
	return nil
}

func NewEndpointSlicesTestPlugin(channelSize int) *endpointSlicesTestPlugin {
	return &endpointSlicesTestPlugin{
		commitCalled: make(chan struct{}, channelSize),
		endpoints:    make(map[string]kapi.Endpoints),
	}
}

func newEndpointSlicesTestController(plugin router.Plugin, initialObjects ...runtime.Object) (*fakekubeclient.Clientset, *RouterController) {
	client := fakekubeclient.NewSimpleClientset(initialObjects...)

	factory := NewDefaultRouterControllerFactory(
		fakerouterclient.NewSimpleClientset(),
		&fakeproject.FakeProjects{},
		client,
		false,
	)

	return client, factory.Create(plugin, false)
}

	stopCh := make(chan struct{})

	return client, factory.Create(plugin, false, stopCh), stopCh
}

// Sort functions should be the inverse of
// endpointsubset.DefaultEndpointAddressOrderByFuncs()
func testEndpointAddressOrderByFuncs() []endpointsubset.EndpointAddressLessFunc {
	ip := func(x, y *kapi.EndpointAddress) bool {
		return !endpointsubset.EndpointAddressIPLessFn(x, y)
	}

	hostname := func(x, y *kapi.EndpointAddress) bool {
		return !endpointsubset.EndpointAddressHostnameLessFn(x, y)
	}

	return []endpointsubset.EndpointAddressLessFunc{
		ip,
		hostname,
	}
}

// Sort functions should be the inverse of
// endpointsubset.DefaultEndpointPortOrderByFuncs()
func testEndpointPortOrderByFuncs() []endpointsubset.EndpointPortLessFunc {
	port := func(x, y *kapi.EndpointPort) bool {
		return !endpointsubset.EndpointPortPortNumberLessFn(x, y)
	}

	protocol := func(x, y *kapi.EndpointPort) bool {
		return !endpointsubset.EndpointPortProtocolLessFn(x, y)
	}

	name := func(x, y *kapi.EndpointPort) bool {
		return !endpointsubset.EndpointPortNameLessFn(x, y)
	}

	return []endpointsubset.EndpointPortLessFunc{
		port,
		protocol,
		name,
	}
}

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}

// stringPtr returns a pointer to the passed string.
func stringPtr(s string) *string {
	return &s
// protocolPtr returns a pointer to the passed protocol.
func protocolPtr(p kapi.Protocol) *kapi.Protocol {
	return &p
}

func TestEndpointSlicesInitialSync(t *testing.T) {
	type testCase struct {
		description         string
		serviceName         string
		expectedCommitCount int
		expectedEventType   watch.EventType
	}

	testCases := []testCase{{
		description:         "without service label, expect no MODIFIED event",
		serviceName:         "",
		expectedCommitCount: 1,
		expectedEventType:   watch.EventType(""),
	}, {
		description:         "with service label, expect MODIFIED event",
		serviceName:         "service-1",
		expectedCommitCount: 1,
		expectedEventType:   watch.Modified,
	}}

	for i, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			plugin := NewEndpointSlicesTestPlugin(100)

			initialObjs := []runtime.Object{
				&discoveryv1beta1.EndpointSlice{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-eps-%d", i),
						Namespace: "test-namespace",
					},
				},
			}

			if len(tc.serviceName) != 0 {
				initialObjs[0].(*discoveryv1beta1.EndpointSlice).Labels = map[string]string{
					ServiceNameLabel: tc.serviceName,
				}
			}

			_, controller, stopCh := newEndpointSlicesTestController(plugin, initialObjs...)
			defer func() {
				close(stopCh)
			}()

			controller.Run()
			if !controller.firstSyncDone {
				t.Fatalf("expected first sync to be completed")
			}

			if err := plugin.WaitForNCommits(tc.expectedCommitCount, endpointSliceTestTimeout); err != nil {
				t.Fatalf("did not receive %v calls to Commit(): %v", tc.expectedCommitCount, err)
			}
		})
	}
}

// Sort functions should be the inverse of
// endpointsubset.DefaultEndpointAddressOrderByFuncs()
func testEndpointAddressOrderByFuncs() []endpointsubset.EndpointAddressLessFunc {
	ip := func(x, y *kapi.EndpointAddress) bool {
		return !endpointsubset.EndpointAddressIPLessFn(x, y)
	}

	hostname := func(x, y *kapi.EndpointAddress) bool {
		return !endpointsubset.EndpointAddressHostnameLessFn(x, y)
	}

	return []endpointsubset.EndpointAddressLessFunc{
		ip,
		hostname,
	}
}

// Sort functions should be the inverse of
// endpointsubset.DefaultEndpointPortOrderByFuncs()
func testEndpointPortOrderByFuncs() []endpointsubset.EndpointPortLessFunc {
	port := func(x, y *kapi.EndpointPort) bool {
		return !endpointsubset.EndpointPortPortNumberLessFn(x, y)
	}

	protocol := func(x, y *kapi.EndpointPort) bool {
		return !endpointsubset.EndpointPortProtocolLessFn(x, y)
	}

	name := func(x, y *kapi.EndpointPort) bool {
		return !endpointsubset.EndpointPortNameLessFn(x, y)
	}

	return []endpointsubset.EndpointPortLessFunc{
		port,
		protocol,
		name,
	}
}

	endpointSlices := []discoveryv1beta1.EndpointSlice{{
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
	}, {
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
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slice-1",
			Namespace: "namespace-b",
			Labels: map[string]string{
				ServiceNameLabel: "service-b",
			},
		},
		AddressType: discoveryv1beta1.AddressTypeIPv4,
		Endpoints: []discoveryv1beta1.Endpoint{{
			Addresses: []string{
				"172.17.0.2",
				"172.17.0.1",
			},
		}},
		Ports: []discoveryv1beta1.EndpointPort{{
			Port: int32Ptr(65000),
		}, {
			Port: int32Ptr(32000),
		}},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slice-1",
			Namespace: "namespace-c",
		},
		AddressType: discoveryv1beta1.AddressTypeIPv4,
		Endpoints: []discoveryv1beta1.Endpoint{{
			Addresses: []string{
				"192.168.30.2",
				"192.168.30.1",
			},
		}},
		Ports: []discoveryv1beta1.EndpointPort{{
			Port: int32Ptr(43),
		}, {
			Port: int32Ptr(42),
		}},
	}}

	plugin := NewEndpointSlicesTestPlugin(100)
	client, controller, stopCh := newEndpointSlicesTestController(plugin)
	defer close(stopCh)

	controller.Run()
	if !controller.firstSyncDone {
		t.Fatalf("expected first sync to be completed")
	}

	for i := 0; i < len(endpointSlices); i++ {
		// Create a copy as we later mutatate the sort order
		// of addresses and ports.
		eps := endpointSlices[i].DeepCopy()
		if _, err := client.DiscoveryV1beta1().EndpointSlices(eps.Namespace).Create(context.TODO(), eps, metav1.CreateOptions{}); err != nil {
			t.Fatalf("failed to create endpointslice %s/%s: %v", eps.Namespace, eps.Name, err)
		}
	}

	if err := plugin.WaitForNCommits(len(endpointSlices), endpointSliceTestTimeout); err != nil {
		t.Fatalf("did not receive %v calls to Commit(): %v", len(endpointSlices), err)
	}

	if expected, actual := watch.Modified, plugin.lastEventType; actual != expected {
		t.Errorf("expected event type %q, got %q", expected, actual)
	}

	type testCase struct {
		description    string
		endpointSlices []discoveryv1beta1.EndpointSlice
		namespace      string
		serviceName    string
		expectService  bool
	}

	testCases := []testCase{{
		description:    "multiple slices, all with service label",
		endpointSlices: endpointSlices[0:2],
		namespace:      "namespace-a",
		serviceName:    "service-a",
		expectService:  true,
	}, {
		description:    "single slice with service label",
		endpointSlices: endpointSlices[2:3],
		namespace:      "namespace-b",
		serviceName:    "service-b",
		expectService:  true,
	}, {
		endpointSlices: endpointSlices[3:],
		description:    "slice with no service label",
		namespace:      "namespace-c",
		serviceName:    "service-c",
		expectService:  false,
	}}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Log(tc.description)

			key := path.Join(tc.namespace, tc.serviceName)
			pluginEndpoints, ok := plugin.endpoints[key]

			if !tc.expectService {
				if ok {
					t.Errorf("did not expect to find service %q", key)
				}
				return
			}

			if !ok {
				t.Fatalf("expected plugin to have been notified for service %q", key)
			}

			sortedEndpointSubsets := convertEndpointSliceToEndpointSubset(tc.endpointSlices, endpointsubset.DefaultEndpointAddressOrderByFuncs(), endpointsubset.DefaultEndpointPortOrderByFuncs())
			unsortedEndpointSubsets := convertEndpointSliceToEndpointSubset(tc.endpointSlices, testEndpointAddressOrderByFuncs(), testEndpointPortOrderByFuncs())

			if diff := cmp.Diff(sortedEndpointSubsets, unsortedEndpointSubsets); diff == "" {
				t.Fatalf("sorting mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(sortedEndpointSubsets, pluginEndpoints.Subsets); diff != "" {
				t.Fatalf("sorting mismatch (-want +got):\n%s", diff)
			}

			if !reflect.DeepEqual(sortedEndpointSubsets, pluginEndpoints.Subsets) {
				t.Fatalf("expected subsets to be equal")
			}
		})
	}
}

}
