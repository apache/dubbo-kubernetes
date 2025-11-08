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
	"io"
	"os"
	"strings"
	"sync"
)

// Interceptor intercepts log output from other logging libraries and reformats it
type Interceptor struct {
	formatter *Formatter
	original  io.Writer
	mu        sync.Mutex
	enabled   bool
}

var (
	globalInterceptor *Interceptor
	interceptorOnce   sync.Once
)

// GetInterceptor returns the global interceptor instance
func GetInterceptor() *Interceptor {
	interceptorOnce.Do(func() {
		globalInterceptor = NewInterceptor()
	})
	return globalInterceptor
}

// NewInterceptor creates a new interceptor
func NewInterceptor() *Interceptor {
	return &Interceptor{
		formatter: GetFormatter(),
		original:  os.Stderr,
		enabled:   true,
	}
}

// Enable enables the interceptor
func (i *Interceptor) Enable() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.enabled = true
}

// Disable disables the interceptor
func (i *Interceptor) Disable() {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.enabled = false
}

// IsEnabled returns whether the interceptor is enabled
func (i *Interceptor) IsEnabled() bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.enabled
}

// Writer returns a writer that intercepts and reformats log output
func (i *Interceptor) Writer() io.Writer {
	return &interceptWriter{interceptor: i}
}

// SetOutput sets the output writer for the interceptor
func (i *Interceptor) SetOutput(w io.Writer) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.original = w
}

// interceptWriter wraps the interceptor to implement io.Writer
type interceptWriter struct {
	interceptor *Interceptor
	buffer      []byte
	mu          sync.Mutex
}

func (w *interceptWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.interceptor.IsEnabled() {
		return w.interceptor.original.Write(p)
	}

	// Append to buffer
	w.buffer = append(w.buffer, p...)

	// Process complete lines
	lines := []string{}
	lineStart := 0
	for i, b := range w.buffer {
		if b == '\n' {
			line := string(w.buffer[lineStart:i])
			// Trim whitespace but keep non-empty lines
			line = strings.TrimSpace(line)
			if line != "" {
				lines = append(lines, line)
			}
			lineStart = i + 1
		}
	}

	// Keep incomplete line in buffer
	if lineStart < len(w.buffer) {
		w.buffer = w.buffer[lineStart:]
	} else {
		w.buffer = w.buffer[:0]
	}

	// Format and write each line (formatter ensures no blank lines)
	for _, line := range lines {
		formatted := w.interceptor.formatter.Format(line)
		// Remove all newlines and add exactly one
		formatted = strings.TrimRight(formatted, "\n\r \t")
		if formatted != "" {
			formatted = formatted + "\n"
			w.interceptor.original.Write([]byte(formatted))
		}
	}

	return len(p), nil
}

// SetupKlogInterceptor sets up klog interception using klog's SetOutput
// This function should be called early in the program initialization
// The actual implementation is in klog_interceptor.go (with build tag)
func SetupKlogInterceptor() {
	// This will be implemented in a separate file that imports klog
	// to avoid forcing all users to have klog as a dependency
}
