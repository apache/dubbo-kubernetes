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
	"bytes"
	"encoding/base32"
)

type Bitset struct {
	bt []byte
}

func (b *Bitset) Set(idx int32) {
	index := idx / 8
	bitsIndex := idx % 8
	for int32(len(b.bt)) <= index {
		b.bt = append(b.bt, 0)
	}
	b.bt[index] |= 0b10000000 >> bitsIndex
}

var base = base32.NewEncoding("1234567890abcdefghijklmnopqrstuv").WithPadding('z')

func (b *Bitset) Encode() string {
	bf := bytes.NewBuffer(nil)
	encoder := base32.NewEncoder(base, bf)
	_, _ = encoder.Write(b.bt)
	_ = encoder.Close()
	return bf.String()
}

func (b *Bitset) Decode(encoded string) error {
	b.bt = []byte{}
	decoded, err := base.DecodeString(encoded)
	if err != nil {
		return err
	}
	b.bt = decoded
	return nil
}

func NewBitset() *Bitset {
	return &Bitset{
		bt: []byte{},
	}
}

func (b *Bitset) Range(f func(idx int, val bool) bool) {
	for i, val := range b.bt {
		for a, mask := 0, byte(0b10000000); a < 8; a, mask = a+1, mask>>1 {
			if !f(i*8+a, val&mask > 0) {
				return
			}
		}
	}
}
