package endpointsubset_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/core/v1"
	discoveryv1beta1 "k8s.io/api/discovery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openshift/router/pkg/router/controller/endpointsubset"
)

// int32Ptr returns a pointer to an int32
func int32Ptr(i int32) *int32 {
	return &i
}

// boolPtr returns a pointer to a bool
func boolPtr(v bool) *bool {
	return &v
}

func TestConvertEndpointSlice(t *testing.T) {
	eps := discoveryv1beta1.EndpointSlice{
		TypeMeta: metav1.TypeMeta{
			Kind:       "endpointslices",
			APIVersion: "discovery.k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "slice-1",
			Namespace: "namespace-a",
			Labels: map[string]string{
				discoveryv1beta1.LabelServiceName: "service-a",
			},
		},
		AddressType: discoveryv1beta1.AddressTypeIPv4,
		Endpoints: []discoveryv1beta1.Endpoint{{
			Addresses: []string{
				"192.168.0.1",
			},
		}},
		Ports: []discoveryv1beta1.EndpointPort{{
			Port: int32Ptr(8080),
		}},
	}

	type args struct {
		items               []discoveryv1beta1.EndpointSlice
		addressOrderByFuncs []endpointsubset.EndpointAddressLessFunc
		portOrderByFuncs    []endpointsubset.EndpointPortLessFunc
	}

	tests := []struct {
		name       string
		args       args
		want       []v1.EndpointSubset
		conditions discoveryv1beta1.EndpointConditions
	}{{
		name: "no Ready condition set, expect zero NotReadyAddresses",
		args: args{
			items:               []discoveryv1beta1.EndpointSlice{*eps.DeepCopy()},
			addressOrderByFuncs: endpointsubset.DefaultEndpointAddressOrderByFuncs(),
			portOrderByFuncs:    endpointsubset.DefaultEndpointPortOrderByFuncs(),
		},
		conditions: discoveryv1beta1.EndpointConditions{
			Ready: nil,
		},
		want: []v1.EndpointSubset{{
			Addresses: []v1.EndpointAddress{{
				IP: "192.168.0.1",
			}},
			NotReadyAddresses: nil,
			Ports: []v1.EndpointPort{{
				Port: 8080,
			}},
		}},
	}, {
		name: "Ready condition set to true, expect zero NotReadyAddresses",
		args: args{
			items:               []discoveryv1beta1.EndpointSlice{*eps.DeepCopy()},
			addressOrderByFuncs: endpointsubset.DefaultEndpointAddressOrderByFuncs(),
			portOrderByFuncs:    endpointsubset.DefaultEndpointPortOrderByFuncs(),
		},
		conditions: discoveryv1beta1.EndpointConditions{
			Ready: boolPtr(true),
		},
		want: []v1.EndpointSubset{{
			Addresses: []v1.EndpointAddress{{
				IP: "192.168.0.1",
			}},
			NotReadyAddresses: nil,
			Ports: []v1.EndpointPort{{
				Port: 8080,
			}},
		}},
	}, {
		name: "Ready condition set to false, expect zero ReadyAddresses and non-zero NotReadyAddresses",
		args: args{
			items:               []discoveryv1beta1.EndpointSlice{*eps.DeepCopy()},
			addressOrderByFuncs: endpointsubset.DefaultEndpointAddressOrderByFuncs(),
			portOrderByFuncs:    endpointsubset.DefaultEndpointPortOrderByFuncs(),
		},
		conditions: discoveryv1beta1.EndpointConditions{
			Ready: boolPtr(false),
		},
		want: []v1.EndpointSubset{{
			Addresses: nil,
			NotReadyAddresses: []v1.EndpointAddress{{
				IP: "192.168.0.1",
			}},
			Ports: []v1.EndpointPort{{
				Port: 8080,
			}},
		}},
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.args.items[0].Endpoints[0].Conditions = tc.conditions
			got := endpointsubset.ConvertEndpointSlice(tc.args.items, tc.args.addressOrderByFuncs, tc.args.portOrderByFuncs)
			if diff := cmp.Diff(got, tc.want); len(diff) != 0 {
				t.Errorf("ConvertEndpointSlice() failed (-want +got):\n%s", diff)
			}
		})
	}
}
