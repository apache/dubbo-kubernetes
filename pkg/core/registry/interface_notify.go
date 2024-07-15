package registry

import (
	"context"

	"dubbo.apache.org/dubbo-go/v3/common"
	dubboRegistry "dubbo.apache.org/dubbo-go/v3/registry"
)

type InterfaceNotifyListener struct {
	NotifyListener
}

func NewInterfaceNotifyListener(listener NotifyListener) *InterfaceNotifyListener {
	return &InterfaceNotifyListener{
		listener,
	}
}

func (l *InterfaceNotifyListener) Notify(event *dubboRegistry.ServiceEvent) {
	l.NotifyListener.Notify(event)
}

func (l *InterfaceNotifyListener) NotifyAll(events []*dubboRegistry.ServiceEvent, f func()) {
	l.NotifyListener.NotifyAll(events, f)
}

func (l *InterfaceNotifyListener) deleteDataplane(ctx context.Context, url *common.URL) error {
	return l.NotifyListener.deleteDataplane(ctx, url)
}

func (l *InterfaceNotifyListener) createOrUpdateDataplane(ctx context.Context, url *common.URL) error {
	return l.NotifyListener.createOrUpdateDataplane(ctx, url)
}
