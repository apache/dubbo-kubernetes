package model

type Controller interface {
	Run(stop <-chan struct{})
	HasSynced() bool
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
