/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Color codes for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorGreen  = "\033[32m"
	ColorBold   = "\033[1m"
)

// PrettyFormatter provides pretty-printed log output with colors
type PrettyFormatter struct {
	useColors   bool
	colorOutput bool
	timeFormat  string
	showScope   bool
	maxScopeLen int
}

var (
	globalPrettyFormatter *PrettyFormatter
	prettyOnce            sync.Once
)

// GetPrettyFormatter returns the global pretty formatter instance
func GetPrettyFormatter() *PrettyFormatter {
	prettyOnce.Do(func() {
		globalPrettyFormatter = NewPrettyFormatter()
	})
	return globalPrettyFormatter
}

// NewPrettyFormatter creates a new pretty formatter
func NewPrettyFormatter() *PrettyFormatter {
	return &PrettyFormatter{
		useColors:   true,
		colorOutput: isColorTerminal(os.Stderr),
		timeFormat:  "2006-01-02 15:04:05.000",
		showScope:   true,
		maxScopeLen: 12,
	}
}

// isColorTerminal checks if the output is a color terminal
func isColorTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return isatty(f.Fd())
	}
	return false
}

// isatty checks if file descriptor is a terminal (simplified version)
func isatty(fd uintptr) bool {
	// Check if TERM environment variable is set (indicates terminal)
	term := os.Getenv("TERM")
	if term == "" {
		return false
	}
	// Check if NO_COLOR is set (disables colors)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Assume it's a terminal if TERM is set and not "dumb"
	return term != "dumb"
}

// SetUseColors enables or disables color output
func (pf *PrettyFormatter) SetUseColors(enable bool) {
	pf.useColors = enable
}

// SetTimeFormat sets the time format for log output
func (pf *PrettyFormatter) SetTimeFormat(format string) {
	pf.timeFormat = format
}

// SetShowScope enables or disables scope display
func (pf *PrettyFormatter) SetShowScope(show bool) {
	pf.showScope = show
}

// SetMaxScopeLength sets the maximum scope name length for alignment
func (pf *PrettyFormatter) SetMaxScopeLength(length int) {
	pf.maxScopeLen = length
}

// Format formats a log line with pretty printing
func (pf *PrettyFormatter) Format(timestamp time.Time, level, scope, message string) string {
	var parts []string

	// Format timestamp
	timeStr := timestamp.Format(pf.timeFormat)
	if pf.useColors && pf.colorOutput {
		timeStr = ColorGray + timeStr + ColorReset
	}
	parts = append(parts, timeStr)

	// Format level with color and icon
	levelStr := pf.formatLevel(level)
	parts = append(parts, levelStr)

	// Format scope
	if pf.showScope {
		scopeStr := pf.formatScope(scope)
		parts = append(parts, scopeStr)
	}

	// Format message (clean any newlines)
	message = strings.TrimRight(message, "\n\r")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.TrimSpace(message)
	messageStr := pf.formatMessage(message)
	parts = append(parts, messageStr)

	return strings.Join(parts, " ")
}

// formatLevel formats the log level with color and icon
func (pf *PrettyFormatter) formatLevel(level string) string {
	var color, icon string

	switch strings.ToLower(level) {
	case "error":
		color = ColorRed
		icon = "✗"
	case "warn":
		color = ColorYellow
		icon = "⚠"
	case "info":
		color = ColorCyan
		icon = "ℹ"
	case "debug":
		color = ColorGray
		icon = "•"
	default:
		color = ColorReset
		icon = "•"
	}

	levelUpper := strings.ToUpper(level)
	// Pad to 5 characters for alignment
	if len(levelUpper) < 5 {
		levelUpper = levelUpper + strings.Repeat(" ", 5-len(levelUpper))
	}

	if pf.useColors && pf.colorOutput {
		return fmt.Sprintf("%s%s%s %s%s", color, ColorBold, icon, levelUpper, ColorReset)
	}
	// Even without colors, show icon for better readability
	return fmt.Sprintf("%s %s", icon, levelUpper)
}

// formatScope formats the scope name with color
func (pf *PrettyFormatter) formatScope(scope string) string {
	scopeStr := scope
	if len(scopeStr) > pf.maxScopeLen {
		scopeStr = scopeStr[:pf.maxScopeLen]
	} else {
		scopeStr = scopeStr + strings.Repeat(" ", pf.maxScopeLen-len(scopeStr))
	}

	if pf.useColors && pf.colorOutput {
		return ColorBlue + ColorBold + scopeStr + ColorReset
	}
	return scopeStr
}

// formatMessage formats the message, preserving any existing formatting
func (pf *PrettyFormatter) formatMessage(message string) string {
	// Check if message contains "repeated" pattern for deduplication summary
	if strings.Contains(message, "(repeated") {
		if pf.useColors && pf.colorOutput {
			// Highlight the repeated count
			parts := strings.Split(message, "(repeated")
			if len(parts) == 2 {
				return parts[0] + ColorYellow + "(repeated" + parts[1] + ColorReset
			}
		}
	}
	return message
}
