package krt

import "go.uber.org/atomic"

type dependency struct {
	id             collectionUID
	collectionName string
	filter         *filter
}

type collectionUID uint64

func GetStop(opts ...CollectionOption) <-chan struct{} {
	o := buildCollectionOptions(opts...)
	return o.stop
}

func buildCollectionOptions(opts ...CollectionOption) collectionOptions {
	c := &collectionOptions{}
	for _, o := range opts {
		o(c)
	}
	if c.stop == nil {
		c.stop = make(chan struct{})
	}
	return *c
}

var globalUIDCounter = atomic.NewUint64(1)

func nextUID() collectionUID {
	return collectionUID(globalUIDCounter.Inc())
}
