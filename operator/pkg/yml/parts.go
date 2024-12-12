package yml

import (
	"bufio"
	"strings"
)

func SplitString(yamlText string) []string {
	out := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(yamlText))
	parts := []string{}
	active := strings.Builder{}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "---") {
			parts = append(parts, active.String())
			active = strings.Builder{}
		} else {
			active.WriteString(line)
			active.WriteString("\n")
		}
	}
	if active.Len() > 0 {
		parts = append(parts, active.String())
	}
	for _, part := range parts {
		part := strings.TrimSpace(part)
		if len(part) > 0 {
			out = append(out, part)
		}
	}
	return out
}
