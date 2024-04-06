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

package registry

import (
	"context"
)

// Registry is the interface that wraps the basic methods of registry.
type Registry interface {
	// ListServices list all services
	ListServices(ctx *context.Context) ([]string, error)
	// ListInstances list all instances of the service
	ListInstances(ctx *context.Context, applicationName string, serviceName string) ([]string, error)

	// TODO add rules
}

type Service struct {
	Ip   string
	Port int
	// TODO add more info?
}
