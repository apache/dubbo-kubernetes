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

package gateway

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TagWatcher is a simplified implementation for Dubbo
// It checks if a Gateway belongs to the current revision
type TagWatcher interface {
	Run(stopCh <-chan struct{})
	HasSynced() bool
	AddHandler(handler TagHandler)
	IsMine(metav1.ObjectMeta) bool
}

// TagHandler is a callback for when the tags revision change.
type TagHandler func(any)
