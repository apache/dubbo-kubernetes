package model

type Controller interface {
	// Run until a signal is received
	Run(stop <-chan struct{})
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
