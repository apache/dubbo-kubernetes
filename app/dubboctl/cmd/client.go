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

package cmd

import (
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/builders/dockerfile"
	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
	config "github.com/apache/dubbo-kubernetes/app/dubboctl/internal/util"
)

// ClientFactory defines a constructor which assists in the creation of a Client
// for use by commands.
// See the NewClient constructor which is the fully populated ClientFactory used
// by commands by default.
// See NewClientFactory which constructs a minimal ClientFactory for use
// during testing.
type ClientFactory func(...dubbo.Option) (*dubbo.Client, func())

func NewClient(options ...dubbo.Option) (*dubbo.Client, func()) {
	var (
		d = newDubboDeployer()
		o = []dubbo.Option{
			dubbo.WithRepositoriesPath(config.RepositoriesPath()),
			dubbo.WithBuilder(dockerfile.NewBuilder()),
			dubbo.WithPusher(dockerfile.NewPusher()),
			dubbo.WithDeployer(d),
		}
	)
	// Client is constructed with standard options plus any additional options
	// which either augment or override the defaults.
	client := dubbo.New(append(o, options...)...)

	cleanup := func() {}
	return client, cleanup
}

func newDubboDeployer() dubbo.Deployer {
	var options []dubbo.DeployerOpt

	return dubbo.NewDeployer(options...)
}
