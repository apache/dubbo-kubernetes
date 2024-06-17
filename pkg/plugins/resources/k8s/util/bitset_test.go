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

package util

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestBitset(t *testing.T) {
	bitset := NewBitset()

	// Testing Set() and RangeConfig()
	bitset.Set(0)
	bitset.Set(5)
	bitset.Set(10)

	ans := map[int]bool{}
	bitset.Range(func(idx int, val bool) bool {
		if val {
			ans[idx] = true
		}
		return true
	})

	// Testing Encode() and Decode()
	encoded := bitset.Encode()
	fmt.Println("Encoded:", encoded)

	newBitset := NewBitset()
	err := newBitset.Decode(encoded)
	if err != nil {
		t.Errorf("Decode error: %v", err)
	}

	newBitset.Range(func(idx int, val bool) bool {
		if val {
			i := ans[idx]
			if !i {
				t.Errorf("bit %v should be present", idx)
			}
			delete(ans, idx)
		}
		return true
	})
	assert.Equal(t, 0, len(ans))
}
