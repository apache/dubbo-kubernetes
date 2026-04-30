package krt_test

import (
	"strconv"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/kube/krt"
)

func BenchmarkKRTFetch(b *testing.B) {
	workloads := make([]testWorkload, 0, 10000)
	for i := 0; i < 10000; i++ {
		version := "v" + strconv.Itoa(i%10)
		workloads = append(workloads, testWorkload{
			Named: krt.Named{Name: "pod-" + strconv.Itoa(i), Namespace: "ns"},
			Labels: map[string]string{
				"app":     "demo",
				"version": version,
			},
			Weight: i,
		})
	}
	collection := krt.NewStaticCollection[testWorkload](nil, workloads)
	byVersion := krt.NewIndex(collection, "version", func(w testWorkload) []string {
		return []string{w.Labels["version"]}
	})
	ctx := krt.TestingDummyContext{}

	b.Run("direct-index-lookup", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			got := byVersion.Lookup("v7")
			if len(got) != 1000 {
				b.Fatalf("got %d workloads", len(got))
			}
		}
	})

	b.Run("fetch-index", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			got := krt.Fetch(ctx, collection, krt.FilterIndex(byVersion, "v7"))
			if len(got) != 1000 {
				b.Fatalf("got %d workloads", len(got))
			}
		}
	})

	b.Run("fetch-label-scan", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			got := krt.Fetch(ctx, collection, krt.FilterLabel(map[string]string{"version": "v7"}))
			if len(got) != 1000 {
				b.Fatalf("got %d workloads", len(got))
			}
		}
	})
}
