package memory

import (
	config2 "github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/sail/pkg/model"
)

type Handler func(config2.Config, config2.Config, model.Event)

// Monitor provides methods of manipulating changes in the config store
type Monitor interface {
	Run(<-chan struct{})
	AppendEventHandler(config2.GroupVersionKind, Handler)
	ScheduleProcessEvent(ConfigEvent)
}

// ConfigEvent defines the event to be processed
type ConfigEvent struct {
	config config2.Config
	old    config2.Config
	event  model.Event
}
