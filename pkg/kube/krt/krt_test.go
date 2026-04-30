package krt_test

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
)

type testWorkload struct {
	krt.Named
	Labels map[string]string
	Weight int
}

func (w testWorkload) GetLabels() map[string]string {
	return w.Labels
}

type testService struct {
	krt.Named
	Selector map[string]string
}

func (s testService) GetLabelSelector() map[string]string {
	return s.Selector
}

type testEndpoint struct {
	krt.Named
	Service  string
	Workload string
}

type testValue struct {
	Name  string
	Value int
}

func (v testValue) ResourceName() string {
	return v.Name
}

func TestFetchFilters(t *testing.T) {
	workloads := krt.NewStaticCollection[testWorkload](nil, []testWorkload{
		{
			Named: krt.Named{Name: "a", Namespace: "ns1"},
			Labels: map[string]string{
				"app":     "demo",
				"version": "v1",
			},
			Weight: 1,
		},
		{
			Named: krt.Named{Name: "b", Namespace: "ns1"},
			Labels: map[string]string{
				"app":     "demo",
				"version": "v2",
			},
			Weight: 2,
		},
		{
			Named: krt.Named{Name: "c", Namespace: "ns2"},
			Labels: map[string]string{
				"app":     "other",
				"version": "v2",
			},
			Weight: 3,
		},
	})
	byVersion := krt.NewIndex(workloads, "version", func(w testWorkload) []string {
		return []string{w.Labels["version"]}
	})

	assertKeys(t, krt.Fetch(krt.TestingDummyContext{}, workloads, krt.FilterName("a", "ns1")), "ns1/a")
	assertKeys(t, krt.Fetch(krt.TestingDummyContext{}, workloads, krt.FilterNamespace("ns1")), "ns1/a", "ns1/b")
	assertKeys(t, krt.Fetch(krt.TestingDummyContext{}, workloads, krt.FilterLabel(map[string]string{"app": "demo"})), "ns1/a", "ns1/b")
	assertKeys(t, krt.Fetch(krt.TestingDummyContext{}, workloads, krt.FilterIndex(byVersion, "v2")), "ns1/b", "ns2/c")
	assertKeys(t, krt.Fetch(krt.TestingDummyContext{}, workloads, krt.FilterGeneric(func(o any) bool {
		return o.(testWorkload).Weight > 1
	})), "ns1/b", "ns2/c")

	services := krt.NewStaticCollection[testService](nil, []testService{
		{Named: krt.Named{Name: "demo", Namespace: "ns1"}, Selector: map[string]string{"app": "demo"}},
		{Named: krt.Named{Name: "all", Namespace: "ns1"}, Selector: map[string]string{}},
		{Named: krt.Named{Name: "other", Namespace: "ns2"}, Selector: map[string]string{"app": "other"}},
	})
	assertKeys(t, krt.Fetch(krt.TestingDummyContext{}, services, krt.FilterSelects(map[string]string{"app": "demo"})), "ns1/all", "ns1/demo")
	assertKeys(t, krt.Fetch(krt.TestingDummyContext{}, services, krt.FilterSelectsNonEmpty(map[string]string{"app": "demo"})), "ns1/demo")
}

func TestManyCollectionRecomputesFromSecondaryFetch(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	workloads := krt.NewStaticCollection[testWorkload](nil, []testWorkload{
		{Named: krt.Named{Name: "a", Namespace: "ns1"}, Labels: map[string]string{"app": "demo"}},
		{Named: krt.Named{Name: "b", Namespace: "ns1"}, Labels: map[string]string{"app": "demo"}},
	}, krt.WithStop(stop))
	services := krt.NewStaticCollection[testService](nil, []testService{
		{Named: krt.Named{Name: "demo", Namespace: "ns1"}, Selector: map[string]string{"app": "demo"}},
	}, krt.WithStop(stop))

	endpoints := krt.NewManyCollection(services, func(ctx krt.HandlerContext, svc testService) []testEndpoint {
		matches := krt.Fetch(ctx, workloads, krt.FilterLabel(svc.Selector))
		out := make([]testEndpoint, 0, len(matches))
		for _, workload := range matches {
			out = append(out, testEndpoint{
				Named:    krt.Named{Name: svc.Name + "-" + workload.Name, Namespace: svc.Namespace},
				Service:  krt.GetKey(svc),
				Workload: krt.GetKey(workload),
			})
		}
		return out
	}, krt.WithStop(stop), krt.WithName("test/endpoints"))

	waitSynced(t, endpoints)
	eventuallyKeys(t, endpoints, "ns1/demo-a", "ns1/demo-b")

	workloads.Reset([]testWorkload{
		{Named: krt.Named{Name: "a", Namespace: "ns1"}, Labels: map[string]string{"app": "demo"}},
		{Named: krt.Named{Name: "b", Namespace: "ns1"}, Labels: map[string]string{"app": "other"}},
		{Named: krt.Named{Name: "c", Namespace: "ns2"}, Labels: map[string]string{"app": "demo"}},
	})

	eventuallyKeys(t, endpoints, "ns1/demo-a", "ns1/demo-c")
}

func TestJoinAndNestedJoinCollection(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	first := krt.NewStaticCollection[testWorkload](nil, []testWorkload{
		{Named: krt.Named{Name: "a", Namespace: "ns1"}},
		{Named: krt.Named{Name: "b", Namespace: "ns1"}, Weight: 1},
	}, krt.WithStop(stop))
	second := krt.NewStaticCollection[testWorkload](nil, []testWorkload{
		{Named: krt.Named{Name: "b", Namespace: "ns1"}, Weight: 2},
		{Named: krt.Named{Name: "c", Namespace: "ns2"}},
	}, krt.WithStop(stop))

	joined := krt.JoinCollection([]krt.Collection[testWorkload]{first, second}, krt.WithStop(stop), krt.WithName("test/join"))
	waitSynced(t, joined)
	assertKeys(t, joined.List(), "ns1/a", "ns1/b", "ns2/c")
	if got := joined.GetKey("ns1/b"); got == nil || got.Weight != 1 {
		t.Fatalf("JoinCollection should keep the first duplicate, got %#v", got)
	}

	merged := krt.JoinWithMergeCollection([]krt.Collection[testWorkload]{first, second}, func(items []testWorkload) *testWorkload {
		out := items[0]
		for _, item := range items[1:] {
			out.Weight += item.Weight
		}
		return &out
	}, krt.WithStop(stop), krt.WithName("test/merged"))
	waitSynced(t, merged)
	assertKeys(t, merged.List(), "ns1/a", "ns1/b", "ns2/c")
	if got := merged.GetKey("ns1/b"); got == nil || got.Weight != 3 {
		t.Fatalf("JoinWithMergeCollection should merge duplicates, got %#v", got)
	}

	collections := krt.NewStaticCollection[krt.Collection[testWorkload]](nil, []krt.Collection[testWorkload]{first, second}, krt.WithStop(stop))
	nested := krt.NestedJoinCollection(collections, krt.WithStop(stop), krt.WithName("test/nested"))
	waitSynced(t, nested)
	assertKeys(t, nested.List(), "ns1/a", "ns1/b", "ns2/c")

	nestedMerged := krt.NestedJoinWithMergeCollection(collections, func(items []testWorkload) *testWorkload {
		out := items[0]
		for _, item := range items[1:] {
			out.Weight += item.Weight
		}
		return &out
	}, krt.WithStop(stop), krt.WithName("test/nested-merged"))
	waitSynced(t, nestedMerged)
	assertKeys(t, nestedMerged.List(), "ns1/a", "ns1/b", "ns2/c")
	if got := nestedMerged.GetKey("ns1/b"); got == nil || got.Weight != 3 {
		t.Fatalf("NestedJoinWithMergeCollection should merge duplicates, got %#v", got)
	}

	third := krt.NewStaticCollection[testWorkload](nil, []testWorkload{
		{Named: krt.Named{Name: "d", Namespace: "ns3"}},
	}, krt.WithStop(stop))
	collections.ConditionalUpdateObject(third)
	eventuallyKeys(t, nested, "ns1/a", "ns1/b", "ns2/c", "ns3/d")
	eventuallyKeys(t, nestedMerged, "ns1/a", "ns1/b", "ns2/c", "ns3/d")
}

func TestRecomputeTriggerInvalidatesDependants(t *testing.T) {
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })

	source := krt.NewStaticCollection[string](nil, []string{"item"}, krt.WithStop(stop))
	trigger := krt.NewRecomputeTrigger(true, krt.WithStop(stop), krt.WithName("test/recompute"))
	external := 1

	values := krt.NewCollection(source, func(ctx krt.HandlerContext, name string) *testValue {
		trigger.MarkDependant(ctx)
		return &testValue{Name: name, Value: external}
	}, krt.WithStop(stop), krt.WithName("test/values"))

	waitSynced(t, values)
	eventuallyValue(t, values, "item", 1)

	external = 2
	trigger.TriggerRecomputation()
	eventuallyValue(t, values, "item", 2)
}

func waitSynced(t *testing.T, syncer krt.Syncer) {
	t.Helper()
	timeout := make(chan struct{})
	timer := time.AfterFunc(2*time.Second, func() { close(timeout) })
	defer timer.Stop()
	if !syncer.WaitUntilSynced(timeout) {
		t.Fatal("collection did not sync before timeout")
	}
}

func eventuallyKeys[T any](t *testing.T, c krt.Collection[T], want ...string) {
	t.Helper()
	eventually(t, func() bool {
		return reflect.DeepEqual(keys(c.List()), want)
	}, func() string {
		return fmt.Sprintf("got keys %v, want %v", keys(c.List()), want)
	})
}

func eventuallyValue(t *testing.T, c krt.Collection[testValue], key string, want int) {
	t.Helper()
	eventually(t, func() bool {
		got := c.GetKey(key)
		return got != nil && got.Value == want
	}, func() string {
		got := c.GetKey(key)
		if got == nil {
			return "got nil value for " + key
		}
		return "got value " + strconv.Itoa(got.Value)
	})
}

func eventually(t *testing.T, ok func() bool, failure func() string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(failure())
}

func assertKeys[T any](t *testing.T, items []T, want ...string) {
	t.Helper()
	got := keys(items)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got keys %v, want %v", got, want)
	}
}

func keys[T any](items []T) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, krt.GetKey(item))
	}
	sort.Strings(out)
	return out
}
