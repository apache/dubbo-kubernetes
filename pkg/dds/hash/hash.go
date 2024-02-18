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

package hash

import (
	"encoding/hex"
	"fmt"
	"hash/fnv"
)

import (
	"k8s.io/apimachinery/pkg/util/rand"

	k8s_strings "k8s.io/utils/strings"
)

func HashedName(mesh, name string, additionalValuesToHash ...string) string {
	return addSuffix(name, hash(append([]string{mesh, name}, additionalValuesToHash...)))
}

func addSuffix(name, hash string) string {
	const hashLength = 1 + 16 // 1 dash plus 8-byte hash that is encoded with hex
	const k8sNameLengthLimit = 253
	shortenName := k8s_strings.ShortenString(name, k8sNameLengthLimit-hashLength)
	return fmt.Sprintf("%s-%s", shortenName, hash)
}

func hash(ss []string) string {
	hasher := fnv.New64a()
	for _, s := range ss {
		_, _ = hasher.Write([]byte(s))
	}
	b := []byte{}
	b = hasher.Sum(b)

	return rand.SafeEncodeString(hex.EncodeToString(b))
}
