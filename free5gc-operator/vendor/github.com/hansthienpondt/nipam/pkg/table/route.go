package table

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"k8s.io/apimachinery/pkg/labels"
)

type Route struct {
	cidr   netip.Prefix
	labels labels.Set
	data   map[string]any
}

type Routes []Route

func NewRoute(cidr netip.Prefix, l map[string]string, d map[string]any) Route {
	var label labels.Set
	var data map[string]any

	if l == nil {
		label = labels.Set{}
	} else {
		label = labels.Set(l)
	}
	if d == nil {
		data = make(map[string]any)
	} else {
		data = d
	}

	return Route{
		cidr:   cidr.Masked(),
		labels: label,
		data:   data,
	}
}

func (r Route) Equal(r2 Route) bool {
	if r.cidr == r2.cidr && labels.Equals(r.labels, r2.labels) {
		return true
	}
	return false
}

func (r Route) String() string {
	return fmt.Sprintf("%s %s", r.cidr.String(), r.labels.String())
}

func (r Route) Prefix() netip.Prefix {
	return r.cidr
}

func (r Route) Labels() labels.Set {
	return r.labels
}

// satisfy the k8s labels.Label interface
func (r Route) Get(label string) string {
	return r.labels.Get(label)
}

// satisfy the k8s labels.Label interface
func (r Route) Has(label string) bool {
	return r.labels.Has(label)
}

func (r Route) Children(rib *RIB) Routes {
	return rib.Children(r.cidr)
}
func (r Route) Parents(rib *RIB) Routes {
	return rib.Parents(r.cidr)
}

func (r Route) GetAvailablePrefixes(rib *RIB) []netip.Prefix {
	return rib.GetAvailablePrefixes(r.cidr)
}
func (r Route) GetAvailablePrefixByBitLen(rib *RIB, b uint8) netip.Prefix {
	return rib.GetAvailablePrefixByBitLen(r.cidr, b)
}

func (r Route) UpdateLabel(label map[string]string) Route {
	r.labels = labels.Merge(r.labels, labels.Set(label))

	return r
}

func (r Route) DeleteLabels() Route {
	r.labels = labels.Set{}

	return r
}

func (r Route) GetData() map[string]any {
	return r.data
}

func (r Route) SetData(d map[string]any) Route {
	r.data = d

	return r
}
func (r Route) DeleteData() Route {
	var d map[string]any = make(map[string]any)
	r.data = d

	return r
}

// Satisfy the json Interface.
func (r Route) MarshalJSON() ([]byte, error) {
	var result map[string]labels.Set = make(map[string]labels.Set)
	result[r.cidr.String()] = r.labels
	return json.Marshal(result)
}

func (r Routes) Len() int           { return len(r) }
func (r Routes) Less(i, j int) bool { return r[i].cidr.Bits() > r[j].cidr.Bits() }
func (r Routes) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

func (r Routes) MarshalJSON() ([]byte, error) {
	var result map[string]labels.Set = make(map[string]labels.Set)

	for _, route := range r {
		result[route.cidr.String()] = route.labels
	}
	return json.Marshal(result)
}
