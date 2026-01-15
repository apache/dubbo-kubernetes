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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogFormat represents the log output format
type LogFormat int

const (
	FormatKlog LogFormat = iota
	FormatStandard
)

var (
	logFormat   LogFormat = FormatStandard // Default to standard format
	logFormatMu sync.RWMutex
	processID   = os.Getpid()
	// callerSkip: Skip frames to get to user code
	// Call stack: runtime.Caller -> formatKlogLine -> Logger.log -> Logger.Info/Infof/etc -> user code
	// So we need to skip 4 frames
	callerSkip = 4
)

// SetLogFormat sets the global log output format
func SetLogFormat(format LogFormat) {
	logFormatMu.Lock()
	defer logFormatMu.Unlock()
	logFormat = format
}

// GetLogFormat returns the current log output format
func GetLogFormat() LogFormat {
	logFormatMu.RLock()
	defer logFormatMu.RUnlock()
	return logFormat
}

func formatKlogLine(level Level, scope, message string, skip int) string {
	now := time.Now()

	// Get level prefix: I, W, E, F
	levelPrefix := levelToKlogPrefix(level)

	// Format timestamp: MMDD HH:MM:SS.microseconds
	monthDay := fmt.Sprintf("%02d%02d", int(now.Month()), now.Day())
	timeStr := fmt.Sprintf("%02d:%02d:%02d.%06d",
		now.Hour(), now.Minute(), now.Second(), now.Nanosecond()/1000)
	timestamp := fmt.Sprintf("%s %s", monthDay, timeStr)

	// Get caller information (file:line)
	file, line := getCaller(skip)

	// Format: I1107 07:53:52.789817 3352004 server.go:652] message
	// Clean message of any newlines
	message = strings.TrimRight(message, "\n\r")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	message = strings.TrimSpace(message)

	// If scope is provided and not empty, include it in the message
	if scope != "" && scope != "default" {
		message = fmt.Sprintf("[%s] %s", scope, message)
	}

	return fmt.Sprintf("%s%s %d %s:%d] %s",
		levelPrefix, timestamp, processID, file, line, message)
}

// levelToKlogPrefix converts log level to klog prefix
func levelToKlogPrefix(level Level) string {
	switch level {
	case ErrorLevel:
		return "E" // Error
	case WarnLevel:
		return "W" // Warning
	case InfoLevel:
		return "I" // Info
	case DebugLevel:
		return "I" // Debug uses Info prefix in klog (V levels are used for debug)
	default:
		return "I"
	}
}

// getCaller returns the file and line number of the caller
func getCaller(skip int) (string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", 0
	}

	// Get just the filename, not the full path
	filename := filepath.Base(file)

	// Try to get function name for better context (optional)
	// This can be useful for debugging
	_ = pc // Can use runtime.FuncForPC(pc).Name() if needed

	return filename, line
}

// Column widths for aligned log output
const (
	levelColumnWidth = 6  // "info  ", "warn  ", "error ", "debug "
	scopeColumnWidth = 12 // Scope names, e.g., "default      "
)

// formatFixedWidthTimestamp formats a timestamp with fixed width (always 9 digits for nanoseconds)
// Format: 2025-11-07T08:07:12.640033925Z (always 30 characters)
func formatFixedWidthTimestamp(t time.Time) string {
	// Format the base part: 2025-11-07T08:07:12.
	base := t.UTC().Format("2006-01-02T15:04:05.")

	// Get nanoseconds and pad to 9 digits
	nanos := t.Nanosecond()
	nanosStr := fmt.Sprintf("%09d", nanos)

	// Append Z for UTC
	return base + nanosStr + "Z"
}

// formatStandardLine formats a log line in standard format with column alignment
// Format: timestamp level scope message (aligned columns, single space between logical fields)
// Example: 2025-11-07T08:02:28.610950444Z info  default      FLAG: --clusterAliases="[]"
// Note: Padding spaces are used for column alignment, but logical fields are separated by single space
func formatStandardLine(timestamp, level, scope, message string) string {
	// Ensure message has no newlines (should already be cleaned by formatMessage, but double-check)
	message = strings.TrimRight(message, "\n\r")
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")
	// Remove any leading/trailing whitespace from message
	message = strings.TrimSpace(message)

	// Align level to fixed width (left-aligned, padded with spaces) for column alignment
	levelPadded := fmt.Sprintf("%-*s", levelColumnWidth, level)

	// Align scope to fixed width (left-aligned, padded with spaces) for column alignment
	scopePadded := fmt.Sprintf("%-*s", scopeColumnWidth, scope)

	// Format: timestamp level scope message
	// Single space after timestamp, then padded level and scope for alignment, then single space before message
	return fmt.Sprintf("%s %s%s %s", timestamp, levelPadded, scopePadded, message)
}
