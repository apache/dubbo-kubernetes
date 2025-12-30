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
	"sync"
	"time"
)

// LogKey represents a unique key for deduplication
type LogKey struct {
	Scope   string
	Level   string
	Message string
}

// DedupEntry represents a log entry with deduplication info
type DedupEntry struct {
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int64
}

// Deduplicator prevents duplicate log messages from flooding the output
type Deduplicator struct {
	entries       map[LogKey]*DedupEntry
	mu            sync.RWMutex
	window        time.Duration // time window for deduplication
	maxCount      int64         // max count before forcing output
	enabled       bool
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

var (
	globalDeduplicator *Deduplicator
	dedupOnce          sync.Once
)

// DefaultDeduplicationWindow is the default time window for deduplication
const DefaultDeduplicationWindow = 5 * time.Second

// DefaultMaxDeduplicationCount is the default max count before forcing output
const DefaultMaxDeduplicationCount = 100

// GetDeduplicator returns the global deduplicator instance
func GetDeduplicator() *Deduplicator {
	dedupOnce.Do(func() {
		globalDeduplicator = NewDeduplicator(DefaultDeduplicationWindow, DefaultMaxDeduplicationCount)
	})
	return globalDeduplicator
}

// NewDeduplicator creates a new deduplicator
func NewDeduplicator(window time.Duration, maxCount int64) *Deduplicator {
	d := &Deduplicator{
		entries:     make(map[LogKey]*DedupEntry),
		window:      window,
		maxCount:    maxCount,
		enabled:     true,
		stopCleanup: make(chan bool),
	}

	// Start cleanup goroutine
	d.cleanupTicker = time.NewTicker(window * 2)
	go d.cleanup()

	return d
}

// ShouldLog checks if a log should be output and updates the deduplication state
// Returns (shouldLog, count, summaryMessage)
func (d *Deduplicator) ShouldLog(key LogKey) (bool, int64, string) {
	if !d.enabled {
		return true, 1, ""
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	entry, exists := d.entries[key]

	if !exists {
		// First time seeing this log
		d.entries[key] = &DedupEntry{
			FirstSeen: now,
			LastSeen:  now,
			Count:     1,
		}
		return true, 1, ""
	}

	// Update entry
	entry.Count++
	entry.LastSeen = now

	// Check if we should output
	timeSinceFirst := now.Sub(entry.FirstSeen)

	// Force output if:
	// 1. Time window has passed
	// 2. Count exceeds max
	if timeSinceFirst >= d.window || entry.Count >= d.maxCount {
		count := entry.Count
		delete(d.entries, key)

		if count > 1 {
			summary := fmt.Sprintf("(repeated %d times in %v)", count, timeSinceFirst)
			return true, count, summary
		}
		return true, 1, ""
	}

	// Suppress duplicate log
	return false, entry.Count, ""
}

// Enable enables deduplication
func (d *Deduplicator) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
}

// Disable disables deduplication
func (d *Deduplicator) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
	// Clear entries when disabled
	d.entries = make(map[LogKey]*DedupEntry)
}

// IsEnabled returns whether deduplication is enabled
func (d *Deduplicator) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

// SetWindow sets the deduplication time window
func (d *Deduplicator) SetWindow(window time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.window = window
	if d.cleanupTicker != nil {
		d.cleanupTicker.Stop()
	}
	d.cleanupTicker = time.NewTicker(window * 2)
}

// SetMaxCount sets the maximum count before forcing output
func (d *Deduplicator) SetMaxCount(maxCount int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxCount = maxCount
}

// cleanup periodically removes old entries
func (d *Deduplicator) cleanup() {
	for {
		select {
		case <-d.cleanupTicker.C:
			d.mu.Lock()
			now := time.Now()
			for key, entry := range d.entries {
				// Remove entries that are older than 2x the window
				if now.Sub(entry.LastSeen) > d.window*2 {
					delete(d.entries, key)
				}
			}
			d.mu.Unlock()
		case <-d.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup goroutine
func (d *Deduplicator) Stop() {
	if d.cleanupTicker != nil {
		d.cleanupTicker.Stop()
	}
	close(d.stopCleanup)
}

// Clear clears all deduplication entries
func (d *Deduplicator) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries = make(map[LogKey]*DedupEntry)
}

// Stats returns deduplication statistics
func (d *Deduplicator) Stats() map[LogKey]int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[LogKey]int64)
	for key, entry := range d.entries {
		stats[key] = entry.Count
	}
	return stats
}
