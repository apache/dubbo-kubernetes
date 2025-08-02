package krt

import (
	"time"
)

func waitForCacheSync(name string, stop <-chan struct{}, collections ...<-chan struct{}) (r bool) {
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()
	_ = time.Now()
	defer func() {
		if r {
		} else {
		}
	}()
	for _, col := range collections {
		for {
			select {
			case <-t.C:
				continue
			case <-stop:
				return false
			case <-col:
			}
			break
		}
	}
	return true
}
