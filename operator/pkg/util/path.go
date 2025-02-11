package util

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	PathSeparator        = "."
	pathSeparatorRune    = '.'
	kvSeparatorRune      = ':'
	InsertIndex          = -1
	EscapedPathSeparator = "\\" + PathSeparator
)

type Path []string

func PathFromString(path string) Path {
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, PathSeparator)
	path = strings.TrimSuffix(path, PathSeparator)
	pv := splitEscaped(path, pathSeparatorRune)
	var r []string
	for _, str := range pv {
		if str != "" {
			str = strings.ReplaceAll(str, EscapedPathSeparator, PathSeparator)
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

func splitEscaped(s string, r rune) []string {
	var prev rune
	if len(s) == 0 {
		return []string{}
	}
	prevIdx := 0
	var out []string
	for i, c := range s {
		if c == r && (i == 0 || (i > 0 && prev != '\\')) {
			out = append(out, s[prevIdx:i])
			prevIdx = i + 1
		}
		prev = c
	}
	out = append(out, s[prevIdx:])
	return out
}

func IsNPathElement(pe string) bool {
	pe, ok := RemoveBrackets(pe)
	if !ok {
		return false
	}

	n, err := strconv.Atoi(pe)
	return err == nil && n >= InsertIndex
}

func RemoveBrackets(pe string) (string, bool) {
	if !strings.HasPrefix(pe, "[") || !strings.HasSuffix(pe, "]") {
		return "", false
	}
	return pe[1 : len(pe)-1], true
}

func IsKVPathElement(pe string) bool {
	pe, ok := RemoveBrackets(pe)
	if !ok {
		return false
	}

	kv := splitEscaped(pe, kvSeparatorRune)
	if len(kv) != 2 || len(kv[0]) == 0 || len(kv[1]) == 0 {
		return false
	}
	return IsValidPathElement(kv[0])
}

func IsValidPathElement(pe string) bool {
	return ValidKeyRegex.MatchString(pe)
}

var ValidKeyRegex = regexp.MustCompile("^[a-zA-Z0-9_-]*$")

func PathN(pe string) (int, error) {
	if !IsNPathElement(pe) {
		return -1, fmt.Errorf("%s is not a valid index path element", pe)
	}
	v, _ := RemoveBrackets(pe)
	return strconv.Atoi(v)
}

func PathKV(pe string) (k, v string, err error) {
	if !IsKVPathElement(pe) {
		return "", "", fmt.Errorf("%s is not a valid key:value path element", pe)
	}
	pe, _ = RemoveBrackets(pe)
	kv := splitEscaped(pe, kvSeparatorRune)
	return kv[0], kv[1], nil
}

func PathV(pe string) (string, error) {
	if IsVPathElement(pe) {
		v, _ := RemoveBrackets(pe)
		return v[1:], nil
	}

	v, _ := RemoveBrackets(pe)
	if len(v) > 0 {
		return v, nil
	}
	return "", fmt.Errorf("%s is not a valid value path element", pe)
}

func IsVPathElement(pe string) bool {
	pe, ok := RemoveBrackets(pe)
	if !ok {
		return false
	}

	return len(pe) > 1 && pe[0] == ':'
}
