//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// LogEntry represents a parsed log entry
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Scope     string
	Message   string
	Original  string
}

// Formatter formats log entries in Istio style
type Formatter struct {
	patterns []*LogPattern
}

// LogPattern represents a pattern for recognizing different log formats
type LogPattern struct {
	Name      string
	Pattern   *regexp.Regexp
	Extractor func([]string) *LogEntry
}

var (
	defaultFormatter *Formatter
	formatterOnce    sync.Once
)

// GetFormatter returns the default formatter instance
func GetFormatter() *Formatter {
	formatterOnce.Do(func() {
		defaultFormatter = NewFormatter()
	})
	return defaultFormatter
}

// NewFormatter creates a new formatter with built-in patterns
func NewFormatter() *Formatter {
	f := &Formatter{
		patterns: []*LogPattern{
			// Klog pattern: I0926 16:53:33.461184       1 controller.go:123] message
			// or: I0926 16:53:33.461184       1 controller.go:123] successfully acquired lease istio-system/istio-namespace-controller-election
			{
				Name:    "klog",
				Pattern: regexp.MustCompile(`^([IWEF])(\d{4} \d{2}:\d{2}:\d{2}\.\d+)\s+\d+\s+[^\s]+\]\s+(.+)$`),
				Extractor: func(matches []string) *LogEntry {
					if len(matches) < 4 {
						return nil
					}
					level := klogLevelToIstio(matches[1])
					timestamp := parseKlogTimestamp(matches[2])
					message := matches[3]
					return &LogEntry{
						Timestamp: timestamp,
						Level:     level,
						Scope:     "klog",
						Message:   message,
						Original:  strings.Join(matches, " "),
					}
				},
			},
			// Standard log pattern: 2025-09-26T16:53:33.461184Z info message
			{
				Name:    "standard",
				Pattern: regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+(\w+)\s+(.+)$`),
				Extractor: func(matches []string) *LogEntry {
					if len(matches) < 4 {
						return nil
					}
					timestamp, _ := time.Parse(time.RFC3339Nano, matches[1])
					return &LogEntry{
						Timestamp: timestamp,
						Level:     matches[2],
						Scope:     "default",
						Message:   matches[3],
						Original:  strings.Join(matches, " "),
					}
				},
			},
			// Logrus pattern: time="2025-09-26T16:53:33Z" level=info msg="message"
			{
				Name:    "logrus",
				Pattern: regexp.MustCompile(`^time="([^"]+)"\s+level=(\w+)\s+msg="([^"]+)"`),
				Extractor: func(matches []string) *LogEntry {
					if len(matches) < 4 {
						return nil
					}
					timestamp, _ := time.Parse(time.RFC3339, matches[1])
					return &LogEntry{
						Timestamp: timestamp,
						Level:     matches[2],
						Scope:     "logrus",
						Message:   matches[3],
						Original:  strings.Join(matches, " "),
					}
				},
			},
			// Pixiu pattern: 2025-12-20T13:54:15.077Z	WARN	cmd/gateway.go:104            	[startGatewayCmd] failed to init logger...
			// Format: timestamp\tLEVEL\tsource\tmessage
			{
				Name:    "pixiu",
				Pattern: regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\t+(\w+)\t+([^\t]+)\t+(.+)$`),
				Extractor: func(matches []string) *LogEntry {
					if len(matches) < 5 {
						return nil
					}
					// Parse timestamp - Pixiu uses variable precision (milliseconds or nanoseconds)
					timestampStr := matches[1]
					var timestamp time.Time
					// Try RFC3339Nano first (for nanosecond precision)
					if t, err := time.Parse(time.RFC3339Nano, timestampStr); err == nil {
						timestamp = t
					} else if t, err := time.Parse(time.RFC3339, timestampStr); err == nil {
						timestamp = t
					} else {
						// Fallback: parse with custom format handling variable precision
						// Remove Z and parse manually
						timestampStr = strings.TrimSuffix(timestampStr, "Z")
						parts := strings.Split(timestampStr, ".")
						if len(parts) == 2 {
							baseTime, _ := time.Parse("2006-01-02T15:04:05", parts[0])
							// Pad fractional seconds to nanoseconds
							fracStr := parts[1]
							for len(fracStr) < 9 {
								fracStr += "0"
							}
							if len(fracStr) > 9 {
								fracStr = fracStr[:9]
							}
							var nanos int
							fmt.Sscanf(fracStr, "%d", &nanos)
							timestamp = baseTime.Add(time.Duration(nanos) * time.Nanosecond).UTC()
						} else {
							timestamp = time.Now().UTC()
						}
					}
					level := strings.ToLower(matches[2])
					source := matches[3]
					message := matches[4]

					// Extract package name from source (e.g., "httpproxy/routerfilter.go:114" -> "httpproxy")
					packageName := ""
					if idx := strings.Index(source, "/"); idx > 0 {
						packageName = source[:idx]
					} else if idx := strings.Index(source, "."); idx > 0 {
						packageName = source[:idx]
					}

					// Prepend package name to message if available
					if packageName != "" {
						message = packageName + "      " + message
					}

					return &LogEntry{
						Timestamp: timestamp,
						Level:     level,
						Scope:     "gateway",
						Message:   message,
						Original:  strings.Join(matches, " "),
					}
				},
			},
		},
	}
	return f
}

// Format formats a log line in Istio style
// Removes any trailing newlines to prevent blank lines
func (f *Formatter) Format(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	// Remove ANSI color codes (escape sequences)
	line = removeANSICodes(line)

	// Remove any newlines from the input line
	line = strings.TrimRight(line, "\n\r")
	line = strings.ReplaceAll(line, "\n", " ")
	line = strings.ReplaceAll(line, "\r", " ")

	// Try to match against known patterns
	for _, pattern := range f.patterns {
		matches := pattern.Pattern.FindStringSubmatch(line)
		if matches != nil {
			entry := pattern.Extractor(matches)
			if entry != nil {
				// Ensure entry message has no newlines
				entry.Message = strings.TrimRight(entry.Message, "\n\r")
				entry.Message = strings.ReplaceAll(entry.Message, "\n", " ")
				entry.Message = strings.ReplaceAll(entry.Message, "\r", " ")
				return f.formatEntry(entry)
			}
		}
	}

	// If no pattern matches, treat as plain message with current time
	now := time.Now().UTC()
	return formatStandardLine(
		formatFixedWidthTimestamp(now),
		"info",
		"unknown",
		line,
	)
}

// formatEntry formats a log entry in standard style with alignment
func (f *Formatter) formatEntry(entry *LogEntry) string {
	return formatStandardLine(
		formatFixedWidthTimestamp(entry.Timestamp),
		entry.Level,
		entry.Scope,
		entry.Message,
	)
}

// klogLevelToIstio converts klog level to Istio level
func klogLevelToIstio(klogLevel string) string {
	switch klogLevel {
	case "I":
		return "info"
	case "W":
		return "warn"
	case "E":
		return "error"
	case "F":
		return "error"
	default:
		return "info"
	}
}

// parseKlogTimestamp parses klog timestamp format (MMDD HH:MM:SS.microseconds)
func parseKlogTimestamp(klogTime string) time.Time {
	now := time.Now()
	// Klog format: MMDD HH:MM:SS.microseconds
	// We need to construct a full timestamp
	parts := strings.Fields(klogTime)
	if len(parts) != 2 {
		return now.UTC()
	}

	datePart := parts[0] // MMDD
	timePart := parts[1] // HH:MM:SS.microseconds

	// Parse time part
	timeParts := strings.Split(timePart, ".")
	if len(timeParts) != 2 {
		return now.UTC()
	}

	hourMinSec := timeParts[0]
	microseconds := timeParts[1]

	// Parse hour:min:sec
	hmsParts := strings.Split(hourMinSec, ":")
	if len(hmsParts) != 3 {
		return now.UTC()
	}

	// Use current year and parse month/day from datePart
	// Klog uses MMDD format, but we need to handle year correctly
	// If the date is in the future (e.g., parsed date > current date), use previous year
	year := now.Year()
	month := 0
	day := 0
	if len(datePart) == 4 {
		_, _ = fmt.Sscanf(datePart, "%02d%02d", &month, &day)
	}

	hour := 0
	min := 0
	sec := 0
	_, _ = fmt.Sscanf(hmsParts[0], "%d", &hour)
	_, _ = fmt.Sscanf(hmsParts[1], "%d", &min)
	_, _ = fmt.Sscanf(hmsParts[2], "%d", &sec)

	// Parse microseconds (pad to nanoseconds)
	micro := 0
	_, _ = fmt.Sscanf(microseconds, "%d", &micro)
	nanos := micro * 1000

	if month == 0 || day == 0 {
		return now.UTC()
	}

	t := time.Date(year, time.Month(month), day, hour, min, sec, nanos, time.UTC)

	// If the parsed time is in the future (more than 1 day ahead), it's likely from previous year
	// This handles cases where klog logs are from a different year
	if t.After(now.Add(24 * time.Hour)) {
		t = time.Date(year-1, time.Month(month), day, hour, min, sec, nanos, time.UTC)
	}

	return t
}

// removeANSICodes removes ANSI escape sequences (color codes) from a string
func removeANSICodes(s string) string {
	// ANSI escape sequence pattern: \033[...m or \x1b[...m
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}
