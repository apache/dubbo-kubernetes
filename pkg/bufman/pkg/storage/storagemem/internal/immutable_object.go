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

// Package internal splits out ImmutableObject into a separate package from storagemem
// to make it impossible to modify ImmutableObject via direct field access.
package internal

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/normalpath"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage/storageutil"
)

// ImmutableObject is an object that contains a path, external path,
// and data that is never modified.
//
// We make this a struct so there is no weirdness with returning a nil interface.
type ImmutableObject struct {
	storageutil.ObjectInfo

	data []byte
}

// NewImmutableObject returns a new ImmutableObject.
//
// path is expected to always be non-empty.
// If externalPath is empty, normalpath.Unnormalize(path) is used.
func NewImmutableObject(
	path string,
	externalPath string,
	data []byte,
) *ImmutableObject {
	if externalPath == "" {
		externalPath = normalpath.Unnormalize(path)
	}
	return &ImmutableObject{
		ObjectInfo: storageutil.NewObjectInfo(path, externalPath),
		data:       data,
	}
}

// Data returns the data.
//
// DO NOT MODIFY.
func (i *ImmutableObject) Data() []byte {
	return i.data
}
