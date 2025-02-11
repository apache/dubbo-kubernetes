package tpath

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/operator/pkg/util"
	"reflect"
	"regexp"
)

type PathContext struct {
	Parent     *PathContext
	KeyToChild interface{}
	Node       interface{}
}

func GetPathContext(root interface{}, path util.Path, createMissing bool) (*PathContext, bool, error) {
	return getPathContext(&PathContext{Node: root}, path, path, createMissing)
}

func getPathContext(nc *PathContext, fullPath, remainPath util.Path, createMissing bool) (*PathContext, bool, error) {
	if len(remainPath) == 0 {
		return nc, true, nil
	}
	pe := remainPath[0]

	if nc.Node == nil {
		if !createMissing {
			return nil, false, fmt.Errorf("node %s is zero", pe)
		}
		if util.IsNPathElement(pe) || util.IsKVPathElement(pe) {
			nc.Node = []interface{}{}
		} else {
			nc.Node = make(map[string]interface{})
		}
	}

	v := reflect.ValueOf(nc.Node)
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	ncNode := v.Interface()

	if lst, ok := ncNode.([]interface{}); ok {
		if util.IsNPathElement(pe) {
			idx, err := util.PathN(pe)
			if err != nil {
				return nil, false, fmt.Errorf("path %s, index %s: %s", fullPath, pe, err)
			}
			var foundNode interface{}
			if idx >= len(lst) || idx < 0 {
				if !createMissing {
					return nil, false, fmt.Errorf("index %d exceeds list length %d at path %s", idx, len(lst), remainPath)
				}
				idx = len(lst)
				foundNode = make(map[string]interface{})
			} else {
				foundNode = lst[idx]
			}
			nn := &PathContext{
				Parent: nc,
				Node:   foundNode,
			}
			nc.KeyToChild = idx
			return getPathContext(nn, fullPath, remainPath[1:], createMissing)
		}
		for idx, le := range lst {
			if lm, ok := le.(map[interface{}]interface{}); ok {
				k, v, err := util.PathKV(pe)
				if err != nil {
					return nil, false, fmt.Errorf("path %s: %s", fullPath, err)
				}
				if stringsEqual(lm[k], v) {
					nn := &PathContext{
						Parent: nc,
						Node:   lm,
					}
					nc.KeyToChild = idx
					nn.KeyToChild = k
					if len(remainPath) == 1 {
						return nn, true, nil
					}
					return getPathContext(nn, fullPath, remainPath[1:], createMissing)
				}
				continue
			}
			// repeat of the block above for the case where tree unmarshals to map[string]interface{}. There doesn't
			// seem to be a way to merge this case into the above block.
			if lm, ok := le.(map[string]interface{}); ok {
				k, v, err := util.PathKV(pe)
				if err != nil {
					return nil, false, fmt.Errorf("path %s: %s", fullPath, err)
				}
				if stringsEqual(lm[k], v) {
					nn := &PathContext{
						Parent: nc,
						Node:   lm,
					}
					nc.KeyToChild = idx
					nn.KeyToChild = k
					if len(remainPath) == 1 {
						return nn, true, nil
					}
					return getPathContext(nn, fullPath, remainPath[1:], createMissing)
				}
				continue
			}
			v, err := util.PathV(pe)
			if err != nil {
				return nil, false, fmt.Errorf("path %s: %s", fullPath, err)
			}
			if matchesRegex(v, le) {
				nn := &PathContext{
					Parent: nc,
					Node:   le,
				}
				nc.KeyToChild = idx
				return getPathContext(nn, fullPath, remainPath[1:], createMissing)
			}
		}
		return nil, false, fmt.Errorf("path %s: element %s not found", fullPath, pe)
	}

	if util.IsMap(ncNode) {
		var nn interface{}
		if m, ok := ncNode.(map[interface{}]interface{}); ok {
			nn, ok = m[pe]
			if !ok {
				if createMissing || len(remainPath) == 1 {
					m[pe] = make(map[interface{}]interface{})
					nn = m[pe]
				} else {
					return nil, false, fmt.Errorf("path not found at element %s in path %s", pe, fullPath)
				}
			}
		}
		if reflect.ValueOf(ncNode).IsNil() {
			ncNode = make(map[string]interface{})
			nc.Node = ncNode
		}
		if m, ok := ncNode.(map[string]interface{}); ok {
			nn, ok = m[pe]
			if !ok {
				if createMissing || len(remainPath) == 1 {
					nextElementNPath := len(remainPath) > 1 && util.IsNPathElement(remainPath[1])
					if nextElementNPath {
						m[pe] = make([]interface{}, 0)
					} else {
						m[pe] = make(map[string]interface{})
					}
					nn = m[pe]
				} else {
					return nil, false, fmt.Errorf("path not found at element %s in path %s", pe, fullPath)
				}
			}
		}

		npc := &PathContext{
			Parent: nc,
			Node:   nn,
		}
		if util.IsSlice(nn) {
			npc.Node = &nn
		}
		nc.KeyToChild = pe
		return getPathContext(npc, fullPath, remainPath[1:], createMissing)
	}

	return nil, false, fmt.Errorf("leaf type %T in non-leaf Node %s", nc.Node, remainPath)
}

func stringsEqual(a, b interface{}) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func matchesRegex(pattern, str interface{}) bool {
	match, err := regexp.MatchString(fmt.Sprint(pattern), fmt.Sprint(str))
	if err != nil {
		fmt.Errorf("bad regex expression %s", fmt.Sprint(pattern))
		return false
	}
	return match
}
