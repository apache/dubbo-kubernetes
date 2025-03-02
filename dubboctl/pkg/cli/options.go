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

package cli

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/pointer"
	"github.com/ory/viper"
	"github.com/spf13/pflag"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	KubeConfigFlag     = "kubeconfig"
	ContextFlag        = "context"
	NamespaceFlag      = "namespace"
	DubboNamespaceFlag = "dubbo-namespace"
)

type RootFlags struct {
	kubeconfig     *string
	Context        *string
	namespace      *string
	dubboNamespace *string
}

func AddRootFlags(flags *pflag.FlagSet) *RootFlags {
	rootFlags := &RootFlags{
		kubeconfig:     pointer.Of[string](""),
		Context:        pointer.Of[string](""),
		namespace:      pointer.Of[string](""),
		dubboNamespace: pointer.Of[string](""),
	}
	flags.StringVarP(rootFlags.kubeconfig, KubeConfigFlag, "c", "", "Kubernetes configuration file")
	flags.StringVar(rootFlags.Context, ContextFlag, "", "Kubernetes configuration context")
	flags.StringVarP(rootFlags.namespace, NamespaceFlag, "n", v1.NamespaceAll, "Kubernetes namespace")
	flags.StringVarP(rootFlags.dubboNamespace, DubboNamespaceFlag, "i", viper.GetString(DubboNamespaceFlag), "Dubbo system namespace")
	return rootFlags
}

func (r *RootFlags) Namespace() string {
	return *r.namespace
}

func (r *RootFlags) DubboNamespace() string {
	return *r.dubboNamespace
}
