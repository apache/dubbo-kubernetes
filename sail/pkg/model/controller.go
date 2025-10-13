package model

import "sync"

type Controller interface {
	Run(stop <-chan struct{})
	HasSynced() bool
}

type ServiceHandler func(*Service, *Service, Event)

type ControllerHandlers struct {
	mutex           sync.RWMutex
	serviceHandlers []ServiceHandler
}

type AggregateController interface {
	Controller
}

type Event int

const (
	EventAdd Event = iota
	EventUpdate
	EventDelete
)

func (event Event) String() string {
	out := "unknown"
	switch event {
	case EventAdd:
		out = "add"
	case EventUpdate:
		out = "update"
	case EventDelete:
		out = "delete"
	}
	return out
}

func (c *ControllerHandlers) NotifyServiceHandlers(prev, curr *Service, event Event) {
	for _, f := range c.GetServiceHandlers() {
		f(prev, curr, event)
	}
}

func (c *ControllerHandlers) GetServiceHandlers() []ServiceHandler {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	// Return a shallow copy of the array
	return c.serviceHandlers
}
