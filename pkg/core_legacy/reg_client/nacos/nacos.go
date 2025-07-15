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

package nacos

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/remoting/nacos"

	nacosClient "github.com/dubbogo/gost/database/kv/nacos"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/extensions"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/core/reg_client"
	"github.com/apache/dubbo-kubernetes/pkg/core/reg_client/factory"
)

func init() {
	mf := &nacosRegClientFactory{}
	extensions.SetRegClientFactory("nacos", func() factory.RegClientFactory {
		return mf
	})
}

type nacosRegClientReport struct {
	client *nacosClient.NacosConfigClient
}

// GetChildren TODO
func (z *nacosRegClientReport) GetChildren(path string) ([]string, error) {
	return nil, nil
}

func (z *nacosRegClientReport) SetContent(path string, value []byte) error {
	return nil
}

func (z *nacosRegClientReport) GetContent(path string) ([]byte, error) {
	return nil, nil
}

func (z *nacosRegClientReport) DeleteContent(path string) error {
	return nil
}

type nacosRegClientFactory struct{}

func (n *nacosRegClientFactory) CreateRegClient(url *common.URL) reg_client.RegClient {
	url.SetParam(constant.NacosNamespaceID, url.GetParam(constant.MetadataReportNamespaceKey, ""))
	url.SetParam(constant.TimeoutKey, url.GetParam(constant.TimeoutKey, constant.DefaultRegTimeout))
	url.SetParam(constant.NacosGroupKey, url.GetParam(constant.MetadataReportGroupKey, constant.ServiceDiscoveryDefaultGroup))
	url.SetParam(constant.NacosUsername, url.Username)
	url.SetParam(constant.NacosPassword, url.Password)
	client, err := nacos.NewNacosConfigClientByUrl(url)
	if err != nil {
		logger.Sugar().Errorf("Could not create nacos metadata report. URL: %s", url.String())
		return nil
	}
	return &nacosRegClientReport{client: client}
}
