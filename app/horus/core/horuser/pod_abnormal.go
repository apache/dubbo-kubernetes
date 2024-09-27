package horuser

import (
	"context"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

func (h *Horuser) PodAbnormalCleanManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.PodAbnormalClean, time.Duration(h.cc.PodAbnormal.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) PodAbnormalClean(ctx context.Context) {
	var wg sync.WaitGroup
	for cn := range h.cc.PodAbnormal.KubeMultiple {
		wg.Add(1)
		go func(clusterName string) {
			defer wg.Done()
		}(cn)
	}
	wg.Wait()
}
