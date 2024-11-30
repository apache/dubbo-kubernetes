package values

import (
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/pointer"
	"path/filepath"
	"sigs.k8s.io/yaml"
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

func GetPathHelper[T any](m Map, name string) T {
	return nil
}
