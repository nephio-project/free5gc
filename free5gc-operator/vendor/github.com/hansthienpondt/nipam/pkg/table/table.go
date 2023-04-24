package table

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"sort"
	"sync"

	"github.com/kentik/patricia"
	"github.com/kentik/patricia/generics_tree"
	"go4.org/netipx"
	"k8s.io/apimachinery/pkg/labels"
)

type RIB struct {
	mu   *sync.RWMutex
	tree *generics_tree.TreeV6[Route]
}

type RIBIterator struct {
	iter *generics_tree.TreeIteratorV6[Route]
}

func NewRIB() *RIB {
	return &RIB{
		mu:   new(sync.RWMutex),
		tree: generics_tree.NewTreeV6[Route](),
	}
}

func (rib *RIB) Add(r Route) error {
	var p patricia.IPv6Address
	_, found := rib.Get(r.Prefix())
	if found {
		return fmt.Errorf("Route %s already exists in the routing table", r.cidr.String())
	}

	if r.cidr.Addr().Is4() {
		p = patricia.NewIPv6Address(netip.AddrFrom16(r.cidr.Addr().As16()).AsSlice(), uint(96+r.cidr.Bits()))
	}
	if r.cidr.Addr().Is6() {
		p = patricia.NewIPv6Address(r.cidr.Addr().AsSlice(), uint(r.cidr.Bits()))
	}
	rib.mu.Lock()
	defer rib.mu.Unlock()
	rib.tree.Set(p, r)
	return nil
}

func (rib *RIB) Set(r Route) error {
	var p patricia.IPv6Address

	if r.cidr.Addr().Is4() {
		p = patricia.NewIPv6Address(netip.AddrFrom16(r.cidr.Addr().As16()).AsSlice(), uint(96+r.cidr.Bits()))
	}
	if r.cidr.Addr().Is6() {
		p = patricia.NewIPv6Address(r.cidr.Addr().AsSlice(), uint(r.cidr.Bits()))
	}
	rib.mu.Lock()
	defer rib.mu.Unlock()
	rib.tree.Set(p, r)
	return nil
}

func (rib *RIB) Delete(r Route) error {
	var p patricia.IPv6Address

	if r.cidr.Addr().Is4() {
		p = patricia.NewIPv6Address(netip.AddrFrom16(r.cidr.Addr().As16()).AsSlice(), uint(96+r.cidr.Bits()))
	}
	if r.cidr.Addr().Is6() {
		p = patricia.NewIPv6Address(r.cidr.Addr().AsSlice(), uint(r.cidr.Bits()))
	}
	matchFunc := func(r1, r2 Route) bool {
		return r1.Equal(r2)
	}
	rib.mu.Lock()
	defer rib.mu.Unlock()
	len := rib.tree.Delete(p, matchFunc, r)
	if len == 0 {
		return fmt.Errorf("Route %s does not exist in the routing table", r.cidr.String())

	}
	return nil
}

func (rib *RIB) Clone() *RIB {
	return &RIB{
		mu:   new(sync.RWMutex),
		tree: rib.tree.Clone(),
	}
}

func (rib *RIB) LPM(cidr netip.Prefix) Routes {
	var p patricia.IPv6Address

	if cidr.Addr().Is4() {
		p = patricia.NewIPv6Address(netip.AddrFrom16(cidr.Addr().As16()).AsSlice(), uint(96+cidr.Bits()))
	}
	if cidr.Addr().Is6() {
		p = patricia.NewIPv6Address(cidr.Addr().AsSlice(), uint(cidr.Bits()))
	}
	// rib.tree.FindTags returns tags from parents
	// rib.tree.FindDeepestTag performs a LPM lookup

	//fmt.Println(rib.tree.FindTagsWithFilter(p, nil))
	rib.mu.RLock()
	defer rib.mu.RUnlock()
	foundLabels := make([]Route, 0)
	foundLabels = rib.tree.FindTagsWithFilterAppend(foundLabels, p, nil)

	return foundLabels
}

func (rib *RIB) Get(cidr netip.Prefix) (Route, bool) {
	// returns true if exact match is found.
	rib.mu.RLock()
	defer rib.mu.RUnlock()

	iter := rib.Iterate()

	for iter.Next() {
		if iter.Route().cidr == cidr {
			return iter.Route(), true
		}
	}
	return Route{cidr: cidr, labels: labels.Set{}}, false
}

func (rib *RIB) GetByLabel(selector labels.Selector) Routes {
	var routes Routes

	rib.mu.RLock()
	defer rib.mu.RUnlock()

	iter := rib.Iterate()

	for iter.Next() {
		if selector.Matches(iter.Route().Labels()) {
			routes = append(routes, iter.Route())
		}
	}

	return routes
}
func (rib *RIB) GetAvailablePrefixes(cidr netip.Prefix) []netip.Prefix {
	var bldr netipx.IPSetBuilder
	bldr.AddPrefix(cidr)

	for _, pfx := range rib.Children(cidr) {
		bldr.RemovePrefix(pfx.Prefix())
	}
	s, err := bldr.IPSet()
	if err != nil {
		return []netip.Prefix{}
	}
	return s.Prefixes()
}

func (rib *RIB) GetAvailablePrefixByBitLen(cidr netip.Prefix, b uint8) netip.Prefix {
	var bldr netipx.IPSetBuilder
	var p netip.Prefix
	bldr.AddPrefix(cidr)

	for _, pfx := range rib.Children(cidr) {
		bldr.RemovePrefix(pfx.Prefix())
	}
	s, err := bldr.IPSet()
	if err != nil {
		return netip.Prefix{}
	}
	p, _, _ = s.RemoveFreePrefix(b)
	return p
}

func (rib *RIB) Iterate() *RIBIterator {
	rib.mu.RLock()
	defer rib.mu.RUnlock()

	return &RIBIterator{
		iter: rib.tree.Iterate()}
}

func (rib *RIB) GetTable() (r Routes) {
	rib.mu.RLock()
	defer rib.mu.RUnlock()

	iter := rib.Iterate()

	for iter.Next() {
		r = append(r, iter.Route())
	}
	return r
}

func (rib *RIB) Children(cidr netip.Prefix) (r Routes) {
	var route Route
	rib.mu.RLock()
	defer rib.mu.RUnlock()

	iter := rib.Iterate()

	for iter.Next() {
		route = iter.Route()
		if route.cidr.Overlaps(cidr) && route.cidr.Bits() > cidr.Bits() {
			r = append(r, route)
		}
	}
	return r
}

func (rib *RIB) Parents(cidr netip.Prefix) (r Routes) {
	var route Route

	rib.mu.RLock()
	defer rib.mu.RUnlock()

	iter := rib.Iterate()

	for iter.Next() {
		route = iter.Route()
		if route.cidr.Overlaps(cidr) && route.cidr.Bits() < cidr.Bits() {
			r = append(r, route)
		}
	}
	sort.Sort(r)
	return r
}

func (rib *RIB) Size() int {
	return rib.tree.CountTags()
}

func (i *RIBIterator) Next() bool {
	return i.iter.Next()
}

func (i *RIBIterator) Route() Route {
	var addrSlice []byte = make([]byte, 16)
	binary.BigEndian.PutUint64(addrSlice, i.iter.Address().Left)
	binary.BigEndian.PutUint64(addrSlice[8:], i.iter.Address().Right)

	l := i.iter.Tags()

	return l[0]
}
