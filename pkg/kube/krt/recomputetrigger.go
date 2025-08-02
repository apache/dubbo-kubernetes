package krt

import (
	"github.com/apache/dubbo-kubernetes/pkg/ptr"
	"go.uber.org/atomic"
)

type RecomputeProtected[T any] struct {
	trigger *RecomputeTrigger
	data    T
}

// Get marks us as dependent on the value and fetches it.
func (c RecomputeProtected[T]) Get(ctx HandlerContext) T {
	c.trigger.MarkDependant(ctx)
	return c.data
}

// TriggerRecomputation tells all dependants to recompute
func (c RecomputeProtected[T]) TriggerRecomputation() {
	c.trigger.TriggerRecomputation()
}

// MarkSynced marks this trigger as ready. Before this is called, dependant collections will be blocked.
// This ensures initial state is populated.
func (c RecomputeProtected[T]) MarkSynced() {
	c.trigger.MarkSynced()
}

// Modify modifies the object and triggers a recompution.
func (c RecomputeProtected[T]) Modify(fn func(*T)) {
	fn(&c.data)
	c.TriggerRecomputation()
}

// AccessUnprotected returns the data without marking as dependant. This must be used with caution; any use within a collection
// is likely broken
func (c RecomputeProtected[T]) AccessUnprotected() T {
	return c.data
}

// NewRecomputeProtected builds a RecomputeProtected which wraps some data, ensuring it is always MarkDependant when accessed
func NewRecomputeProtected[T any](initialData T, startSynced bool, opts ...CollectionOption) RecomputeProtected[T] {
	trigger := NewRecomputeTrigger(startSynced, opts...)
	return RecomputeProtected[T]{
		trigger: trigger,
		data:    initialData,
	}
}

// RecomputeTrigger trigger provides an escape hatch to allow krt transformations to depend on external state and recompute
// correctly when those change.
// Typically, all state is registered and fetched through krt.Fetch. Through this mechanism, any changes are automatically
// propagated through the system to dependencies.
// In some cases, it may not be feasible to get all state into krt; hopefully, this is a temporary state.
// RecomputeTrigger works around this by allowing an explicit call to recompute a collection; the caller must be sure to call Trigger()
// any time the state changes.
type RecomputeTrigger struct {
	inner StaticSingleton[int32]
	// krt will suppress events for unchanged resources. To workaround this, we constantly change and int each time TriggerRecomputation
	// is called to ensure our event is not suppressed.
	i *atomic.Int32
}

func NewRecomputeTrigger(startSynced bool, opts ...CollectionOption) *RecomputeTrigger {
	inner := NewStatic[int32](ptr.Of(int32(0)), startSynced, opts...)
	return &RecomputeTrigger{inner: inner, i: atomic.NewInt32(0)}
}

// TriggerRecomputation tells all dependants to recompute
func (r *RecomputeTrigger) TriggerRecomputation() {
	v := r.i.Inc()
	r.inner.Set(ptr.Of(v))
}

// MarkDependant marks the given context as depending on this trigger. This registers it to be recomputed when TriggerRecomputation
// is called.
func (r *RecomputeTrigger) MarkDependant(ctx HandlerContext) {
	_ = Fetch(ctx, r.inner.AsCollection())
}

// MarkSynced marks this trigger as ready. Before this is called, dependant collections will be blocked.
// This ensures initial state is populated.
func (r *RecomputeTrigger) MarkSynced() {
	r.inner.MarkSynced()
}
