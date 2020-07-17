package endpointsubset

import (
	"sort"

	kapi "k8s.io/api/core/v1"
)

type EndpointAddressLessFunc func(x, y *kapi.EndpointAddress) bool

type EndpointAddressMultiSorter struct {
	addresses []kapi.EndpointAddress
	less      []EndpointAddressLessFunc
}

var _ sort.Interface = &EndpointAddressMultiSorter{}

// Sort sorts the argument slice according to the comparator functions
// passed to orderBy.
func (s *EndpointAddressMultiSorter) Sort(addresses []kapi.EndpointAddress) {
	s.addresses = addresses
	sort.Sort(s)
}

// NewEndpointAddressOrderBy returns a Sorter that sorts using a number
// of comparator functions.
func NewEndpointAddressOrderBy(less ...EndpointAddressLessFunc) *EndpointAddressMultiSorter {
	return &EndpointAddressMultiSorter{
		less: less,
	}
}

// Len is part of sort.Interface.
func (s *EndpointAddressMultiSorter) Len() int {
	return len(s.addresses)
}

// Swap is part of sort.Interface.
func (s *EndpointAddressMultiSorter) Swap(i, j int) {
	s.addresses[i], s.addresses[j] = s.addresses[j], s.addresses[i]
}

// Less is part of sort.Interface.
func (s *EndpointAddressMultiSorter) Less(i, j int) bool {
	p, q := s.addresses[i], s.addresses[j]

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

func EndpointAddressDefaultSortFields() []EndpointAddressLessFunc {
	hostname := func(x, y *kapi.EndpointAddress) bool {
		return x.Hostname < y.Hostname
	}

	ip := func(x, y *kapi.EndpointAddress) bool {
		return x.IP < y.IP
	}

	// TODO(frobware): what about EndpointAddress.Nodename?
	return []EndpointAddressLessFunc{hostname, ip}
}

func SortAddresses(addresses []kapi.EndpointAddress, comparators []EndpointAddressLessFunc) {
	NewEndpointAddressOrderBy(comparators...).Sort(addresses)
}
