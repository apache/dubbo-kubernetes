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

package dubbo_cp

import (
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/config/dds/debounce"
	"github.com/apache/dubbo-kubernetes/pkg/config/webhook"

	// nolint
	dubbogo "dubbo.apache.org/dubbo-go/v3/config"
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/dds"
	"github.com/pkg/errors"

	"github.com/apache/dubbo-kubernetes/pkg/config/admin"
	"github.com/apache/dubbo-kubernetes/pkg/config/kube"
	"github.com/apache/dubbo-kubernetes/pkg/config/security"
	"github.com/apache/dubbo-kubernetes/pkg/config/server"
)

type Config struct {
	Admin      admin.Admin             `yaml:"admin"`
	GrpcServer server.ServerConfig     `yaml:"grpcServer"`
	Security   security.SecurityConfig `yaml:"security"`
	KubeConfig kube.KubeConfig         `yaml:"kubeConfig"`
	Webhook    webhook.Webhook         `yaml:"webhook"`
	Dubbo      dubbogo.RootConfig      `yaml:"dubbo"`
	Dds        dds.Dds                 `yaml:"dds"`
}

func (c *Config) Sanitize() {
	c.Security.Sanitize()
	c.Admin.Sanitize()
	c.Webhook.Sanitize()
	c.GrpcServer.Sanitize()
	c.KubeConfig.Sanitize()
	c.Dds.Sanitize()
}

func (c *Config) Validate() error {
	err := c.Webhook.Validate()
	if err != nil {
		return errors.Wrap(err, "Webhook validation failed")
	}
	err = c.Security.Validate()
	if err != nil {
		return errors.Wrap(err, "SecurityConfig validation failed")
	}
	err = c.Admin.Validate()
	if err != nil {
		return errors.Wrap(err, "Admin validation failed")
	}
	err = c.GrpcServer.Validate()
	if err != nil {
		return errors.Wrap(err, "ServerConfig validation failed")
	}
	err = c.KubeConfig.Validate()
	if err != nil {
		return errors.Wrap(err, "KubeConfig validation failed")
	}
	err = c.Dds.Validate()
	if err != nil {
		return errors.Wrap(err, "options validation failed")
	}
	return nil
}

var DefaultConfig = func() Config {
	return Config{
		Admin: admin.Admin{
			AdminPort:    38080,
			ConfigCenter: "zookeeper://127.0.0.1:2181",
			MetadataReport: admin.AddressConfig{
				Address: "zookeeper://127.0.0.1:2181",
			},
			Registry: admin.AddressConfig{
				Address: "zookeeper://127.0.0.1:2181",
			},
			Prometheus: admin.Prometheus{
				Address:     "127.0.0.1:9090",
				MonitorPort: "22222",
			},
			Grafana: admin.Grafana{
				Address: "127.0.0.1:93030",
			},
		},
		GrpcServer: server.ServerConfig{
			PlainServerPort:  30060,
			SecureServerPort: 30062,
			DebugPort:        30070,
		},
		Security: security.SecurityConfig{
			CaValidity:           30 * 24 * 60 * 60 * 1000,
			CertValidity:         1 * 60 * 60 * 1000,
			IsTrustAnyone:        false,
			EnableOIDCCheck:      true,
			ResourceLockIdentity: config.GetStringEnv("POD_NAME", config.GetDefaultResourceLockIdentity()),
		},
		Webhook: webhook.Webhook{
			Port:       30080,
			AllowOnErr: true,
		},
		KubeConfig: kube.KubeConfig{
			Namespace:       "dubbo-system",
			ServiceName:     "dubbo-cp",
			RestConfigQps:   50,
			RestConfigBurst: 100,
			KubeFileConfig:  "",
			DomainSuffix:    "cluster.local",
		},
		Dubbo: dubbogo.RootConfig{},
		Dds: dds.Dds{
			Debounce: debounce.Debounce{
				After:  100 * time.Millisecond,
				Max:    10 * time.Second,
				Enable: true,
			},
			SendTimeout: 5 * time.Second,
		},
	}
}
