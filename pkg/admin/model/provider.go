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
	"fmt"
	"sort"
	"strings"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/admin/constant"
)

type Provider struct {
	Entity
	Service        string        `json:"service"`
	URL            string        `json:"url"`
	Parameters     string        `json:"parameters"`
	Address        string        `json:"address"`
	Registry       string        `json:"registry"`
	Dynamic        bool          `json:"dynamic"`
	Enabled        bool          `json:"enabled"`
	Timeout        int64         `json:"timeout"`
	Serialization  string        `json:"serialization"`
	Weight         int64         `json:"weight"`
	Application    string        `json:"application"`
	Username       string        `json:"username"`
	Expired        time.Duration `json:"expired"`
	Alived         int64         `json:"alived"`
	RegistrySource string        `json:"registrySource"`
}

func (p *Provider) InitByUrl(id string, url *common.URL) {
	if url == nil {
		return
	}

	mapToString := func(params map[string]string) string {
		pairs := make([]string, 0, len(params))
		for key, val := range params {
			pairs = append(pairs, fmt.Sprintf("%s=%s", key, val))
		}
		sort.Strings(pairs)
		return strings.Join(pairs, "&")
	}

	p.Entity = Entity{Hash: id}
	p.Service = url.ServiceKey()
	p.Address = url.Location
	p.Application = url.GetParam(constant.ApplicationKey, "")
	p.URL = url.String()
	p.Parameters = mapToString(url.ToMap())
	p.Dynamic = url.GetParamBool(constant.DynamicKey, true)
	p.Enabled = url.GetParamBool(constant.EnabledKey, true)
	p.Serialization = url.GetParam(constant.SerializationKey, "hessian2")
	p.Timeout = url.GetParamInt(constant.TimeoutKey, constant.DefaultTimeout)
	p.Weight = url.GetParamInt(constant.WeightKey, constant.DefaultWeight)
	p.Username = url.GetParam(constant.OwnerKey, "")
	p.RegistrySource = url.GetParam(constant.RegistryType, constant.RegistryInterface)
}
