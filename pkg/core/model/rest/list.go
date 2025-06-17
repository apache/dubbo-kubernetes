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

package rest

import (
	"encoding/json"
)

import (
	"github.com/pkg/errors"
)

import (
	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
)

type ResourceList struct {
	Total uint32     `json:"total"`
	Items []Resource `json:"items"`
	Next  *string    `json:"next"`
}

type ResourceListReceiver struct {
	ResourceList
	NewResource func() core_model.Resource
}

var _ json.Unmarshaler = &ResourceListReceiver{}

func (rec *ResourceListReceiver) UnmarshalJSON(data []byte) error {
	if rec.NewResource == nil {
		return errors.Errorf("NewResource must not be nil")
	}
	type List struct {
		Total uint32             `json:"total"`
		Items []*json.RawMessage `json:"items"`
		Next  *string            `json:"next"`
	}
	list := List{}
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}
	rec.ResourceList.Total = list.Total
	rec.ResourceList.Items = make([]Resource, len(list.Items))
	for i, li := range list.Items {
		b, err := json.Marshal(li)
		if err != nil {
			return err
		}

		restResource := From.Resource(rec.NewResource())
		if err := json.Unmarshal(b, restResource); err != nil {
			return err
		}

		rec.ResourceList.Items[i] = restResource
	}
	rec.ResourceList.Next = list.Next
	return nil
}
