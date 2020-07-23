package endpointsubset

import (
	"sort"

	kapi "k8s.io/api/core/v1"
)

func SortEndpointSubsets(subsets []kapi.EndpointSubset) (sortErr error) {
	sort.SliceStable(subsets, func(i, j int) bool {
		if len(subsets[i].Addresses) == 0 && len(subsets[j].Addresses) == 0 {
			return false
		}
		if len(subsets[i].Addresses) == 0 || len(subsets[j].Addresses) == 0 {
			return len(subsets[i].Addresses) == 0
		}

		for x := range subsets[i].Addresses {
			if subsets[i].Addresses[x].IP == subsets[j].Addresses[x].IP {
				continue
			}
			if subsets[i].Addresses[x].IP < subsets[j].Addresses[x].IP {
				return true
			}
		}

		if len(subsets[i].Ports) == 0 && len(subsets[j].Ports) == 0 {
			return false
		}
		if len(subsets[i].Ports) == 0 || len(subsets[j].Ports) == 0 {
			return len(subsets[i].Ports) == 0
		}

		for x := range subsets[i].Ports {
			if subsets[i].Ports[x].Port == subsets[j].Ports[x].Port {
				continue
			}
			if subsets[i].Ports[x].Port < subsets[j].Ports[x].Port {
				return true
			}
		}
		return false
	})

	// sort.SliceStable(subsets, func(i, j int) bool {
	// 	a, err := json.Marshal(subsets[i].Addresses)
	// 	if sortErr == nil && err != nil {
	// 		sortErr = err
	// 		return false
	// 	}

	// 	b, err := json.Marshal(subsets[i].Addresses)
	// 	if sortErr == nil && err != nil {
	// 		sortErr = err
	// 		return false
	// 	}

	// 	switch bytes.Compare(a, b) {
	// 	case -1:
	// 		return true
	// 	case 1:
	// 		return false
	// 	}

	// 	a, _ = json.Marshal(subsets[i].Ports)
	// 	if sortErr == nil && err != nil {
	// 		sortErr = err
	// 		return false
	// 	}

	// 	b, _ = json.Marshal(subsets[j].Ports)
	// 	if sortErr == nil && err != nil {
	// 		sortErr = err
	// 		return false
	// 	}

	// 	return bytes.Compare(a, b) < 0
	// })

	return
}
