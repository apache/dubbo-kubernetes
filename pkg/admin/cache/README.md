# Development Guide

## Overview
- cache module
  - cache.go: define cache interface and result model.
  - registry:
    - kube
      - cache.go: implement Cache interface for kubernetes mode, and define some registry logic like startInformer, stopInformer, etc.
      - registry.go: implement Registry interface for kubernetes mode, and refer registry logic defined in cache.go
    - universal:
      - cache.go: implement Cache interface for universal mode, and define some registry logic like store, delete, etc.
      - registry.go: implement Registry interface for universal mode, and refer registry logic defined in cache.go
    - extension.go: define Registry extension interface and use it in admin module's bootstrap process.
  - selector:
    - selector.go: define Selector interface and Options interface.
    - application_selector.go: implement Selector interface, and define application selector logic.
    - service_selector.go: implement Selector interface, and define service selector logic.
    - multiple_selector.go: an implement of Selector to combine multiple selectors.

## How to use
- After dubbo-cp setup, cache has been initialized and an instance of Cache is declared as a global var in admin/config.
- Use `config.Cache` to get cache instance.
- Call some methods of Cache to get data from cache.

## Examples

### Get resources by application

```go
package service

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/cache/selector"
	"github.com/apache/dubbo-kubernetes/pkg/admin/config"
)

func (s *XXXServiceImpl) GetXXX(application string) ([]*model.XXX, error) {
	// get data from cache
	xxx, err := config.Cache.GetXXXWithSelector("some-namespace", selector.NewApplicationSelector(application))
	if err != nil {
		return nil, err
	}
	// use data to do something

	// return results
	return yyy, nil
}
```