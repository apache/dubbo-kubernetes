// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zk

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/registry"
)

type zkRegistry struct {
	client *zk.Conn
}

// NewZkRegistry creates a new zookeeper registry
func NewZkRegistry(addr string) (registry.Registry, error) {
	addrList := strings.Split(addr, ",")
	conn, _, err := zk.Connect(addrList, 5*time.Second, zk.WithLogInfo(false))
	if err != nil {
		return nil, err
	}
	return &zkRegistry{
		client: conn,
	}, nil
}

func (z *zkRegistry) ListServices(ctx *context.Context) ([]string, error) {
	children, _, err := z.client.Children("/")
	if err != nil {
		return nil, err
	}
	// if contain provider, then it is a service
	var services []string
	for _, firstChild := range children {
		// first level is application name, like "dubbo"
		secondChildren, _, err := z.client.Children("/" + firstChild)
		if err != nil {
			continue
		}
		for _, secondChild := range secondChildren {
			// second level is service name, like "com.example.user.UserProvider"
			thirdChildren, _, err := z.client.Children("/" + firstChild + "/" + secondChild)
			if err != nil {
				continue
			}
			// third level is "providers" or "consumers"
			for _, thirdChild := range thirdChildren {
				if strings.Contains(thirdChild, "providers") {
					seviceName := fmt.Sprintf("application: %s, service: %s", firstChild, secondChild)
					services = append(services, seviceName)
					break
				}
			}
		}

	}
	return services, nil
}

func (z *zkRegistry) ListInstances(ctx *context.Context, applicationName string, serviceName string) ([]string, error) {
	// get all instances of the service
	children, _, err := z.client.Children("/" + applicationName + "/" + serviceName)
	if err != nil {
		return nil, err
	}
	// get all instances
	var instances []string
	for _, child := range children {
		if strings.EqualFold(child, "providers") {
			serviceInstance, _, err := z.client.Children("/" + applicationName + "/" + serviceName + "/" + child)
			if err != nil {
				continue
			}
			for _, instance := range serviceInstance {
				queryUnescape, err := url.QueryUnescape(instance)
				if err != nil {
					continue
				}
				u, err := url.Parse(queryUnescape)
				if err != nil {
					continue
				}
				instances = append(instances, u.Host)
			}
		}
	}
	return instances, nil
}
