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

package governance

import (
	"errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"

	"github.com/dubbogo/go-zookeeper/zk"
)

const group = "dubbo"

type RuleExists struct {
	cause error
}

func (exist *RuleExists) Error() string {
	return exist.cause.Error()
}

type RuleNotFound struct {
	cause error
}

func (notFound *RuleNotFound) Error() string {
	return notFound.cause.Error()
}

type GovernanceConfig interface {
	SetConfig(key string, value string) error
	GetConfig(key string) (string, error)
	DeleteConfig(key string) error
	SetConfigWithGroup(group string, key string, value string) error
	GetConfigWithGroup(group string, key string) (string, error)
	DeleteConfigWithGroup(group string, key string) error
	Register(url *common.URL) error
	UnRegister(url *common.URL) error
}

var impls map[string]func(cc config_center.DynamicConfiguration, registry registry.Registry) GovernanceConfig

func init() {
	impls = map[string]func(cc config_center.DynamicConfiguration, registry registry.Registry) GovernanceConfig{
		"zookeeper": func(cc config_center.DynamicConfiguration, registry registry.Registry) GovernanceConfig {
			gc := &GovernanceConfigImpl{
				configCenter:   cc,
				registryCenter: registry,
			}
			return &ZkGovImpl{
				GovernanceConfig: gc,
				configCenter:     cc,
				group:            group,
			}
		},
		"nacos": func(cc config_center.DynamicConfiguration, registry registry.Registry) GovernanceConfig {
			gc := &GovernanceConfigImpl{
				configCenter:   cc,
				registryCenter: registry,
			}
			return &NacosGovImpl{
				GovernanceConfig: gc,
				configCenter:     cc,
				group:            group,
			}
		},
	}
}

func NewGovernanceConfig(cc config_center.DynamicConfiguration, registry registry.Registry, p string) GovernanceConfig {
	return impls[p](cc, registry)
}

type GovernanceConfigImpl struct {
	registryCenter registry.Registry
	configCenter   config_center.DynamicConfiguration
}

func (g *GovernanceConfigImpl) SetConfig(key string, value string) error {
	return g.SetConfigWithGroup(group, key, value)
}

func (g *GovernanceConfigImpl) GetConfig(key string) (string, error) {
	return g.GetConfigWithGroup(group, key)
}

func (g *GovernanceConfigImpl) DeleteConfig(key string) error {
	return g.DeleteConfigWithGroup(group, key)
}

func (g *GovernanceConfigImpl) SetConfigWithGroup(group string, key string, value string) error {
	if key == "" || value == "" {
		return errors.New("key or value is empty")
	}
	return g.configCenter.PublishConfig(key, group, value)
}

func (g *GovernanceConfigImpl) GetConfigWithGroup(group string, key string) (string, error) {
	if key == "" {
		return "", errors.New("key is empty")
	}
	return g.configCenter.GetRule(key, config_center.WithGroup(group))
}

func (g *GovernanceConfigImpl) DeleteConfigWithGroup(group string, key string) error {
	if key == "" {
		return errors.New("key is empty")
	}
	return g.configCenter.RemoveConfig(key, group)
}

func (g *GovernanceConfigImpl) Register(url *common.URL) error {
	if url.String() == "" {
		return errors.New("url is empty")
	}
	return g.registryCenter.Register(url)
}

func (g *GovernanceConfigImpl) UnRegister(url *common.URL) error {
	if url.String() == "" {
		return errors.New("url is empty")
	}
	return g.registryCenter.UnRegister(url)
}

type ZkGovImpl struct {
	GovernanceConfig
	configCenter config_center.DynamicConfiguration
	group        string
}

// GetConfig transform ZK specified 'node does not exist' err into unified admin rule error
func (c *ZkGovImpl) GetConfig(key string) (string, error) {
	if key == "" {
		return "", errors.New("key is empty")
	}
	rule, err := c.configCenter.GetRule(key, config_center.WithGroup(c.group))
	if err != nil {
		if errors.Is(err, zk.ErrNoNode) {
			return "", nil
		}
		return "", err
	}
	return rule, nil
}

// SetConfig transform ZK specified 'node already exist' err into unified admin rule error
func (c *ZkGovImpl) SetConfig(key string, value string) error {
	if key == "" || value == "" {
		return errors.New("key or value is empty")
	}
	err := c.configCenter.PublishConfig(key, c.group, value)
	if err != nil {
		if errors.Is(err, zk.ErrNoNode) {
			return nil
		}
		return err
	}
	return nil
}

type NacosGovImpl struct {
	GovernanceConfig
	configCenter config_center.DynamicConfiguration
	group        string
}

// GetConfig transform Nacos specified 'node does not exist' err into unified admin rule error
func (n *NacosGovImpl) GetConfig(key string) (string, error) {
	return n.GovernanceConfig.GetConfig(key)
}

// SetConfig transform Nacos specified 'node already exist' err into unified admin rule error
func (n *NacosGovImpl) SetConfig(key string, value string) error {
	return n.GovernanceConfig.SetConfig(key, value)
}
