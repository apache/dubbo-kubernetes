package log

import (
	"fmt"
	"strings"
	"sync"
)

type Scope struct {
	name string
}

var (
	lock sync.RWMutex
)

func RegisterScope(name string, desc string) *Scope {
	return registerScope(name, desc)
}

func registerScope(name string, desc string) *Scope {
	if strings.ContainsAny(name, ":,.") {
		panic(fmt.Sprintf("scope name %s is invalid, it cannot contain colons, commas, or periods", name))
	}
	return nil
}
