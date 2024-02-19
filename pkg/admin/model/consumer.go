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

package model

import (
	"time"

	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
)

type Consumer struct {
	Entity
	Service     string        `json:"service"`
	Parameters  string        `json:"parameters"`
	Result      string        `json:"result"`
	Address     string        `json:"address"`
	Registry    string        `json:"registry"`
	Application string        `json:"application"`
	Username    string        `json:"username"`
	Statistics  string        `json:"statistics"`
	Collected   time.Duration `json:"collected"`
	Expired     time.Duration `json:"expired"`
	Alived      int64         `json:"alived"`
}

func (c *Consumer) InitByUrl(id string, url *common.URL) {
	if url == nil {
		return
	}

	c.Entity = Entity{Hash: id}
	c.Service = url.ServiceKey()
	c.Address = url.Location
	c.Application = url.GetParam(constant.ApplicationKey, "")
	c.Parameters = url.String()
	c.Username = url.GetParam(constant.OwnerKey, "")
}
