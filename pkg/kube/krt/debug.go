package krt

import (
	"encoding/json"
	"sync"
)

// DebugHandler allows attaching a variety of collections to it and then dumping them
type DebugHandler struct {
	debugCollections []DebugCollection
	mu               sync.RWMutex
}

func (p *DebugHandler) MarshalJSON() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return json.Marshal(p.debugCollections)
}

var GlobalDebugHandler = new(DebugHandler)

type CollectionDump struct {
	// Map of output key -> output
	Outputs map[string]any `json:"outputs,omitempty"`
	// Name of the input collection
	InputCollection string `json:"inputCollection,omitempty"`
	// Map of input key -> info
	Inputs map[string]InputDump `json:"inputs,omitempty"`
	// Synced returns whether the collection is synced or not
	Synced bool `json:"synced"`
}
type InputDump struct {
	Outputs      []string `json:"outputs,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}
type DebugCollection struct {
	name string
	dump func() CollectionDump
	uid  collectionUID
}

func (p DebugCollection) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"uid":   p.uid,
		"name":  p.name,
		"state": p.dump(),
	})
}

// nolint: unused // (not true, not sure why it thinks it is!)
func eraseMap[T any](l map[Key[T]]T) map[string]any {
	nm := make(map[string]any, len(l))
	for k, v := range l {
		nm[string(k)] = v
	}
	return nm
}
