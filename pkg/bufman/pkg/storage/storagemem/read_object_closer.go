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

package storagemem

import (
	"bytes"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage/storagemem/internal"
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage/storageutil"
)

type readObjectCloser struct {
	storageutil.ObjectInfo

	reader *bytes.Reader
	closed bool
}

func newReadObjectCloser(immutableObject *internal.ImmutableObject) *readObjectCloser {
	return &readObjectCloser{
		ObjectInfo: immutableObject.ObjectInfo,
		reader:     bytes.NewReader(immutableObject.Data()),
	}
}

func (r *readObjectCloser) Read(p []byte) (int, error) {
	if r.closed {
		return 0, storage.ErrClosed
	}
	return r.reader.Read(p)
}

func (r *readObjectCloser) Close() error {
	if r.closed {
		return storage.ErrClosed
	}
	r.closed = true
	return nil
}
