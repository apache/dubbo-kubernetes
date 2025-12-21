package krt

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/kube/controllers"
	"github.com/apache/dubbo-kubernetes/pkg/util/ptr"
)

func NewStatusManyCollection[I controllers.Object, IStatus, O any](
	c Collection[I],
	hf TransformationMultiStatus[I, IStatus, O],
	opts ...CollectionOption,
) (Collection[ObjectWithStatus[I, IStatus]], Collection[O]) {
	o := buildCollectionOptions(opts...)
	if o.name == "" {
		o.name = fmt.Sprintf("NewStatusManyCollection[%v,%v,%v]", ptr.TypeName[I](), ptr.TypeName[IStatus](), ptr.TypeName[O]())
	}
	statusOpts := append(opts, WithName(o.name+"/status"))
	statusCh := make(chan struct{})
	statusSynced := channelSyncer{
		name:   o.name + " status",
		synced: statusCh,
	}
	status := NewStaticCollection[ObjectWithStatus[I, IStatus]](statusSynced, nil, statusOpts...)
	// When input is deleted, the transformation function wouldn't run.
	// So we need to handle that explicitly
	cleanupOnRemoval := func(i []Event[I]) {
		for _, e := range i {
			if e.Event == controllers.EventDelete {
				status.DeleteObject(GetKey(e.Latest()))
			}
		}
	}
	primary := newManyCollection[I, O](c, func(ctx HandlerContext, i I) []O {
		st, objs := hf(ctx, i)
		// Create/delete our status objects
		if st == nil {
			status.DeleteObject(GetKey(i))
		} else {
			cs := ObjectWithStatus[I, IStatus]{
				Obj:    i,
				Status: *st,
			}
			status.ConditionalUpdateObject(cs)
		}
		return objs
	}, o, cleanupOnRemoval)
	go func() {
		// Status collection is not synced until the parent is synced.
		// We could almost just pass the primary.Syncer(), but that is circular so doesn't really work.
		if primary.WaitUntilSynced(o.stop) {
			close(statusCh)
		}
	}()

	return status, primary
}

func NewStatusCollection[I controllers.Object, IStatus, O any](
	c Collection[I],
	hf TransformationSingleStatus[I, IStatus, O],
	opts ...CollectionOption,
) (Collection[ObjectWithStatus[I, IStatus]], Collection[O]) {
	hm := func(ctx HandlerContext, i I) (*IStatus, []O) {
		status, res := hf(ctx, i)
		if res == nil {
			return status, nil
		}
		return status, []O{*res}
	}
	return NewStatusManyCollection(c, hm, opts...)
}

type StatusCollection[I controllers.Object, IStatus any] = Collection[ObjectWithStatus[I, IStatus]]

type ObjectWithStatus[I controllers.Object, IStatus any] struct {
	Obj    I
	Status IStatus
}

func (c ObjectWithStatus[I, IStatus]) ResourceName() string {
	return GetKey(c.Obj)
}

func (c ObjectWithStatus[I, IStatus]) Equals(o ObjectWithStatus[I, IStatus]) bool {
	return Equal(c.Obj, o.Obj) && Equal(c.Status, o.Status)
}
