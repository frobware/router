package endpointsubset

import (
	"sort"

	kapi "k8s.io/api/core/v1"
)

type EndpointPortLessFunc func(x, y *kapi.EndpointPort) bool

type EndpointPortMultiSorter struct {
	ports []kapi.EndpointPort
	less  []EndpointPortLessFunc
}

var _ sort.Interface = &EndpointPortMultiSorter{}

// Sort sorts the argument slice according to the comparator functions
// passed to orderBy.
func (s *EndpointPortMultiSorter) Sort(ports []kapi.EndpointPort) {
	s.ports = ports
	sort.Sort(s)
}

// EndpointPortOrderBy returns a Sorter that sorts using a number
// of comparator functions.
func EndpointPortOrderBy(less ...EndpointPortLessFunc) *EndpointPortMultiSorter {
	return &EndpointPortMultiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (s *EndpointPortMultiSorter) Len() int {
	return len(s.ports)
}

// Swap is part of sort.Interface.
func (s *EndpointPortMultiSorter) Swap(i, j int) {
	s.ports[i], s.ports[j] = s.ports[j], s.ports[i]
}

// Less is part of sort.Interface.
func (s *EndpointPortMultiSorter) Less(i, j int) bool {
	p, q := s.ports[i], s.ports[j]

	// Try all but the last comparison.
	var k int
	for k = 0; k < len(s.less)-1; k++ {
		less := s.less[k]
		switch {
		case less(&p, &q):
			// p < q, so we have a decision.
			return true
		case less(&q, &p):
			// p > q, so we have a decision.
			return false
		}
		// p == q; try the next comparison.
	}

	return s.less[k](&p, &q)
}

func EndpointPortDefaultSortFields() []EndpointPortLessFunc {
	name := func(x, y *kapi.EndpointPort) bool {
		return x.Name < y.Name
	}

	portNumber := func(x, y *kapi.EndpointPort) bool {
		return x.Port < y.Port
	}

	protocol := func(x, y *kapi.EndpointPort) bool {
		return x.Protocol < y.Protocol
	}

	return []EndpointPortLessFunc{name, portNumber, protocol}
}

func SortPorts(ports []kapi.EndpointPort, comparators []EndpointPortLessFunc) {
	EndpointPortOrderBy(comparators...).Sort(ports)
}
