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

package pusher

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime/component"
)

// Pusher 's job is to push resource
type Pusher interface {
	component.Component
	// AddCallback add callback for target resource type using id
	// for example, id is a unique id for every client, when resource changed for target resourceType, it will invoke callback
	AddCallback(resourceType core_model.ResourceType, id string, callback ResourceChangedCallbackFn, filters ...ResourceChangedEventFilter)
	// RemoveCallback remove callback
	RemoveCallback(resourceType core_model.ResourceType, id string)
	// InvokeCallback invoke a target callback
	// for example, for a push request from client, invoke this function to push resource.
	InvokeCallback(resourceType core_model.ResourceType, id string, request interface{}, requestFilter ResourceRequestFilter)
}
