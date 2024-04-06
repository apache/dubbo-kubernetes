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

package once

import (
	"sync"
)

type once struct {
	syncOnce sync.Once
	Value    interface{}
	Err      error
}

func (o *once) Do(f func() (interface{}, error)) {
	o.syncOnce.Do(func() {
		o.Value, o.Err = f()
	})
}

func newMap() *omap {
	return &omap{
		m: map[string]*once{},
	}
}

type omap struct {
	mtx sync.Mutex
	m   map[string]*once
}

func (c *omap) Get(key string) (*once, bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	o, exist := c.m[key]
	if !exist {
		o = &once{}
		c.m[key] = o
		return o, true
	}
	return o, false
}

func (c *omap) Delete(key string) {
	c.mtx.Lock()
	delete(c.m, key)
	c.mtx.Unlock()
}
