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

package zookeeper

import (
	"strings"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"

	"github.com/dubbogo/go-zookeeper/zk"

	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/core/consts"
	"github.com/apache/dubbo-kubernetes/pkg/core/extensions"
	"github.com/apache/dubbo-kubernetes/pkg/core/reg_client"
	"github.com/apache/dubbo-kubernetes/pkg/core/reg_client/factory"
)

func init() {
	mf := &zookeeperRegClientFactory{}
	extensions.SetRegClientFactory("zookeeper", func() factory.RegClientFactory {
		return mf
	})
}

type zookeeperRegClient struct {
	client *gxzookeeper.ZookeeperClient
}

func (z *zookeeperRegClient) GetChildren(path string) ([]string, error) {
	children, err := z.client.GetChildren(path)
	if err != nil {
		if errors.Is(err, zk.ErrNoNode) {
			return []string{}, nil
		}
		return nil, err
	}
	return children, nil
}

func (z *zookeeperRegClient) SetContent(path string, value []byte) error {
	err := z.client.CreateWithValue(path, value)
	if err != nil {
		if errors.Is(err, zk.ErrNoNode) {
			return nil
		}
		if errors.Is(err, zk.ErrNodeExists) {
			_, stat, _ := z.client.GetContent(path)
			_, setErr := z.client.SetContent(path, value, stat.Version)
			if setErr != nil {
				return errors.WithStack(setErr)
			}
			return nil
		}
		return errors.WithStack(err)
	}
	return nil
}

func (z *zookeeperRegClient) GetContent(path string) ([]byte, error) {
	content, _, err := z.client.GetContent(path)
	if err != nil {
		if errors.Is(err, zk.ErrNoNode) {
			return []byte{}, nil
		}
		return []byte{}, errors.WithStack(err)
	}
	return content, nil
}

func (z *zookeeperRegClient) DeleteContent(path string) error {
	err := z.client.Delete(path)
	if err != nil {
		if errors.Is(err, zk.ErrNoNode) {
			return nil
		}
		return errors.WithStack(err)
	}
	return nil
}

type zookeeperRegClientFactory struct{}

func (mf *zookeeperRegClientFactory) CreateRegClient(url *common.URL) reg_client.RegClient {
	client, err := gxzookeeper.NewZookeeperClient(
		"zookeeperRegClient",
		strings.Split(url.Location, ","),
		false,
		gxzookeeper.WithZkTimeOut(url.GetParamDuration(consts.TimeoutKey, "25s")),
	)
	if err != nil {
		panic(err)
	}

	return &zookeeperRegClient{client: client}
}
