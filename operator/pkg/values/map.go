package values

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/pointer"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
)

type Map map[string]any

func (m Map) JSON() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("json Marshal: %v", err))
	}
	return string(bytes)
}

func (m Map) YAML() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("yaml Marshal: %v", err))
	}
	return string(bytes)
}

func MapFromJSON(input []byte) (Map, error) {
	m := make(Map)
	err := json.Unmarshal(input, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func MapFromYAML(input []byte) (Map, error) {
	m := make(Map)
	err := json.Unmarshal(input, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func fromJSON[T any](overlay []byte) (T, error) {
	v := new(T)
	err := json.Unmarshal(overlay, &v)
	if err != nil {
		return pointer.Empty[T](), err
	}
	return *v, nil
}

func fromYAML[T any](overlay []byte) (T, error) {
	v := new(T)
	err := yaml.Unmarshal(overlay, &v)
	if err != nil {
		return pointer.Empty[T](), err
	}
	return *v, nil
}

func parsePath(key string) []string { return strings.Split(key, ".") }

func tableLookup(m Map, simple string) (Map, bool) {
	v, ok := m[simple]
	if !ok {
		return nil, false
	}
	if vv, ok := v.(map[string]interface{}); ok {
		return vv, true
	}
	if vv, ok := v.(Map); ok {
		return vv, true
	}
	return nil, false
}

// GetPathMap key.subkey
func (m Map) GetPathMap(s string) (Map, bool) {
	current := m
	for _, n := range parsePath(s) {
		subkey, ok := tableLookup(current, n)
		if !ok {
			return nil, false
		}
		current = subkey
	}
	return current, true
}

func splitEscaped(s string, r rune) []string {
	var prev rune
	if len(s) == 0 {
		return []string{}
	}
	prevIndex := 0
	var out []string
	for i, c := range s {
		if c == r && (i == 0 || i > 0 && prev != '\\') {
			out = append(out, s[prevIndex:i])
			prevIndex = i + 1
		}
		prev = c
	}
	out = append(out, s[prevIndex:])
	return out
}

func splitPath(path string) []string {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, ".")
	path = strings.TrimSuffix(path, ".")
	pv := splitEscaped(path, '.')
	var r []string
	for _, str := range pv {
		if str != "" {
			str = strings.ReplaceAll(str, "\\.", ".")
			nBracket := strings.IndexRune(str, '[')
			if nBracket > 0 {
				r = append(r, str[:nBracket], str[nBracket:])
			} else {
				r = append(r, str)
			}
		}
	}
	return r
}

func (m Map) GetPathString(s string) string {
	return ""
}

func GetPathHelper[T any](m Map, name string) T {
	return nil
}

func (m Map) GetPath(name string) (any, bool) {
	current := any(m)
	paths := splitPath(name)
	for _, n := range paths {
		if idx, ok := extractIndex(n); ok {
			a, ok := current.([]any)
			if !ok {
				return nil, false
			}
			if idx >= 0 && idx < len(a) {
				current = a[idx]
			} else {
				return nil, false
			}
		} else if k, v, ok := extractKeyValue(n); ok {
			a, ok := current.([]any)
			if !ok {
				return nil, false
			}
			index := -1
			for idx, cm := range a {
				if MustCastAsMap(cm)[k] == v {
					index = idx
					break
				}
			}
			if index == -1 {
				return nil, false
			}
			current = a[idx]
		} else {
			cm, ok := CastAsMap(current)
			if !ok {
				return nil, false
			}
			subKey, ok := cm[n]
			if !ok {
				return nil, false
			}
			current = subKey
		}
	}
	if p, ok := current.(*any); ok {
		return *p, true
	}
	return current, true
}

func MustCastAsMap(current any) Map {
	m, ok := CastAsMap(current)
	if !ok {
		if !reflect.ValueOf(current).IsValid() {
			return Map{}
		}
		panic(fmt.Sprintf("not a map, got %T: %v %v", current, current, reflect.ValueOf(current).Kind()))
	}
	return m
}

func CastAsMap(current any) (Map, bool) {
	if m, ok := current.(Map); ok {
		return m, true
	}
	if m, ok := current.(map[string]any); ok {
		return m, true
	}
	return nil, false
}

func extractIndex(seg string) (int, bool) {
	if !strings.HasPrefix(seg, "[") || !strings.HasSuffix(seg, "]") {
		return 0, false
	}
	sanitized := seg[1 : len(seg)-1]
	v, err := strconv.Atoi(sanitized)
	if err != nil {
		return 0, false
	}
	return v, true
}

func extractKeyValue(seg string) (string, string, bool) {
	if !strings.HasPrefix(seg, "[") || !strings.HasSuffix(seg, "]") {
		return "", "", false
	}
	sanitized := seg[1 : len(seg)-1]
	return strings.Cut(sanitized, ":")
}
