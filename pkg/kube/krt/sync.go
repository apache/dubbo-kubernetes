package krt

import "github.com/apache/dubbo-kubernetes/pkg/kube"

type Syncer interface {
	WaitUntilSynced(stop <-chan struct{}) bool
	HasSynced() bool
}

type pollSyncer struct {
	name string
	f    func() bool
}

type multiSyncer struct {
	syncers []Syncer
}

type channelSyncer struct {
	name   string
	synced <-chan struct{}
}

var (
	_ Syncer = channelSyncer{}
	_ Syncer = pollSyncer{}
	_ Syncer = multiSyncer{}
)

func (c channelSyncer) WaitUntilSynced(stop <-chan struct{}) bool {
	return waitForCacheSync(c.name, stop, c.synced)
}

func (c channelSyncer) HasSynced() bool {
	select {
	case <-c.synced:
		return true
	default:
		return false
	}
}

func (c pollSyncer) WaitUntilSynced(stop <-chan struct{}) bool {
	return kube.WaitForCacheSync(c.name, stop, c.f)
}

func (c pollSyncer) HasSynced() bool {
	return c.f()
}

func (c multiSyncer) WaitUntilSynced(stop <-chan struct{}) bool {
	for _, s := range c.syncers {
		if !s.WaitUntilSynced(stop) {
			return false
		}
	}
	return true
}

func (c multiSyncer) HasSynced() bool {
	for _, s := range c.syncers {
		if !s.HasSynced() {
			return false
		}
	}
	return true
}
