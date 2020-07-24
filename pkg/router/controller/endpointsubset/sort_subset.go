package endpointsubset

import (
	"sort"

	kapi "k8s.io/api/core/v1"
)

func SortSubset(subset []kapi.EndpointSubset) {
	sort.SliceStable(subset, func(i, j int) bool {
		// TODO(frobware)

		if len(subset[i].Addresses) == 0 && len(subset[j].Addresses) == 0 {
			return false
		}

		if len(subset[i].Addresses) == 0 || len(subset[j].Addresses) == 0 {
			return len(subset[i].Addresses) == 0
		}

		if len(subset[i].Ports) == 0 && len(subset[j].Ports) == 0 {
			return false
		}

		if len(subset[i].Ports) == 0 || len(subset[j].Ports) == 0 {
			return len(subset[i].Ports) == 0
		}

		return false
	})
}
