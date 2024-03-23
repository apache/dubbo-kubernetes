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

package intercp

import (
	"time"
)

import (
	mesh_proto "github.com/apache/dubbo-kubernetes/api/mesh/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/intercp/client"
	"github.com/apache/dubbo-kubernetes/pkg/intercp/envoyadmin"
)

var log = core.Log.WithName("inter-cp")

func Setup(rt runtime.Runtime) error {
	return nil
}

func DefaultClientPool() *client.Pool {
	return client.NewPool(client.New, 5*time.Minute, core.Now)
}

func PooledEnvoyAdminClientFn(pool *client.Pool) envoyadmin.NewClientFn {
	return func(url string) (mesh_proto.InterCPEnvoyAdminForwardServiceClient, error) {
		conn, err := pool.Client(url)
		if err != nil {
			return nil, err
		}
		return mesh_proto.NewInterCPEnvoyAdminForwardServiceClient(conn), nil
	}
}
