package krt

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/kube/kclient"
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
)

type informer[I controllers.ComparableObject] struct {
	inf            kclient.Informer[I]
	collectionName string
	id             collectionUID
	augmentation   func(a any) any
	synced         chan struct{}
	baseSyncer     Syncer
	metadata       Metadata
}

type channelSyncer struct {
	name   string
	synced <-chan struct{}
}

func WrapClient[I controllers.ComparableObject](c kclient.Informer[I], opts ...CollectionOption) Collection[I] {
	o := buildCollectionOptions(opts...)
	if o.name == "" {
		o.name = fmt.Sprintf("Informer[%v]", ptr.TypeName[I]())
	}
	h := &informer[I]{
		inf:            c,
		collectionName: o.name,
		id:             nextUID(),
		augmentation:   o.augmentation,
		synced:         make(chan struct{}),
	}
	h.baseSyncer = channelSyncer{
		name:   h.collectionName,
		synced: h.synced,
	}

	if o.metadata != nil {
		h.metadata = o.metadata
	}
	return nil
}

func (c channelSyncer) WaitUntilSynced(stop <-chan struct{}) bool {
	return waitForCacheSync(c.name, stop, c.synced)
}
