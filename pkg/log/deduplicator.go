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

type LogKey struct {
	Scope   string
	Level   string
	Message string
}

type DedupEntry struct {
	FirstSeen time.Time
	LastSeen  time.Time
	Count     int64
}

type Deduplicator struct {
	entries       map[LogKey]*DedupEntry
	mu            sync.RWMutex
	window        time.Duration
	maxCount      int64
	enabled       bool
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

var (
	globalDeduplicator *Deduplicator
	dedupOnce          sync.Once
)

const DefaultDeduplicationWindow = 5 * time.Second

const DefaultMaxDeduplicationCount = 100

func GetDeduplicator() *Deduplicator {
	dedupOnce.Do(func() {
		globalDeduplicator = NewDeduplicator(DefaultDeduplicationWindow, DefaultMaxDeduplicationCount)
	})
	return globalDeduplicator
}

func NewDeduplicator(window time.Duration, maxCount int64) *Deduplicator {
	d := &Deduplicator{
		entries:     make(map[LogKey]*DedupEntry),
		window:      window,
		maxCount:    maxCount,
		enabled:     true,
		stopCleanup: make(chan bool),
	}

	d.cleanupTicker = time.NewTicker(window * 2)
	go d.cleanup()

	return d
}

func (d *Deduplicator) ShouldLog(key LogKey) (bool, int64, string) {
	if !d.enabled {
		return true, 1, ""
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	entry, exists := d.entries[key]

	if !exists {
		d.entries[key] = &DedupEntry{
			FirstSeen: now,
			LastSeen:  now,
			Count:     1,
		}
		return true, 1, ""
	}

	entry.Count++
	entry.LastSeen = now

	timeSinceFirst := now.Sub(entry.FirstSeen)

	if timeSinceFirst >= d.window || entry.Count >= d.maxCount {
		count := entry.Count
		delete(d.entries, key)

		if count > 1 {
			summary := fmt.Sprintf("(repeated %d times in %v)", count, timeSinceFirst)
			return true, count, summary
		}
		return true, 1, ""
	}

	return false, entry.Count, ""
}

func (d *Deduplicator) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
}

func (d *Deduplicator) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
	d.entries = make(map[LogKey]*DedupEntry)
}

func (d *Deduplicator) IsEnabled() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.enabled
}

func (d *Deduplicator) SetWindow(window time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.window = window
	if d.cleanupTicker != nil {
		d.cleanupTicker.Stop()
	}
	d.cleanupTicker = time.NewTicker(window * 2)
}

func (d *Deduplicator) SetMaxCount(maxCount int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxCount = maxCount
}

func (d *Deduplicator) cleanup() {
	for {
		select {
		case <-d.cleanupTicker.C:
			d.mu.Lock()
			now := time.Now()
			for key, entry := range d.entries {
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

func (d *Deduplicator) Stop() {
	if d.cleanupTicker != nil {
		d.cleanupTicker.Stop()
	}
	close(d.stopCleanup)
}

func (d *Deduplicator) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.entries = make(map[LogKey]*DedupEntry)
}

func (d *Deduplicator) Stats() map[LogKey]int64 {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := make(map[LogKey]int64)
	for key, entry := range d.entries {
		stats[key] = entry.Count
	}
	return stats
}
