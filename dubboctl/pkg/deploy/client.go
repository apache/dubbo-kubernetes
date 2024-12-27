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

package deploy

//import (
//	dubbo2 "github.com/apache/dubbo-kubernetes/operator/dubbo"
//	dubbohttp "github.com/apache/dubbo-kubernetes/operator/http"
//	"github.com/apache/dubbo-kubernetes/operator/pkg/builders/pack"
//	"github.com/apache/dubbo-kubernetes/operator/pkg/docker"
//	"github.com/apache/dubbo-kubernetes/operator/pkg/docker/creds"
//	"github.com/apache/dubbo-kubernetes/operator/pkg/docker/prompt"
//	config "github.com/apache/dubbo-kubernetes/operator/pkg/util"
//	"net/http"
//	"os"
//)
//
//// ClientFactory defines a constructor which assists in the creation of a Client
//// for use by commands.
//// See the NewClient constructor which is the fully populated ClientFactory used
//// by commands by default.
//// See NewClientFactory which constructs a minimal ClientFactory for use
//// during testing.
//type ClientFactory func(...dubbo2.Option) (*dubbo2.Client, func())
//
//func NewClient(options ...dubbo2.Option) (*dubbo2.Client, func()) {
//	var (
//		t = newTransport(false)
//		c = newCredentialsProvider(config.Dir(), t)
//		d = newDubboDeployer()
//		o = []dubbo2.Option{
//			dubbo2.WithRepositoriesPath(config.RepositoriesPath()),
//			dubbo2.WithBuilder(pack.NewBuilder()),
//			dubbo2.WithPusher(docker.NewPusher(
//				docker.WithCredentialsProvider(c),
//				docker.WithTransport(t))),
//			dubbo2.WithDeployer(d),
//		}
//	)
//	// Client is constructed with standard options plus any additional options
//	// which either augment or override the defaults.
//	client := dubbo2.New(append(o, options...)...)
//
//	cleanup := func() {}
//	return client, cleanup
//}
//
//func newDubboDeployer() dubbo2.Deployer {
//	var options []dubbo2.DeployerOpt
//
//	return dubbo2.NewDeployer(options...)
//}
//
//// newTransport returns a transport with cluster-flavor-specific variations
//// which take advantage of additional features offered by cluster variants.
//func newTransport(insecureSkipVerify bool) dubbohttp.RoundTripCloser {
//	return dubbohttp.NewRoundTripper(dubbohttp.WithInsecureSkipVerify(insecureSkipVerify))
//}
//
//// newCredentialsProvider returns a credentials provider which possibly
//// has cluster-flavor specific additional credential loaders to take advantage
//// of features or configuration nuances of cluster variants.
//func newCredentialsProvider(configPath string, t http.RoundTripper) docker.CredentialsProvider {
//	options := []creds.Opt{
//		creds.WithPromptForCredentials(prompt.NewPromptForCredentials(os.Stdin, os.Stdout, os.Stderr)),
//		creds.WithPromptForCredentialStore(prompt.NewPromptForCredentialStore()),
//		creds.WithTransport(t),
//	}
//
//	// Other cluster variants can be supported here
//	return creds.NewCredentialsProvider(configPath, options...)
//}
