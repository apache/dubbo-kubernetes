package krt

import "github.com/apache/dubbo-kubernetes/pkg/util/smallset"

type filter struct {
	keys smallset.Set[string]
}

func FilterKey(k string) FetchOption {
	return func(h *dependency) {
		h.filter.keys = smallset.New(k)
	}
}
