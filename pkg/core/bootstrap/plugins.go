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

package bootstrap

import (
	_ "dubbo.apache.org/dubbo-go/v3/config_center/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/zookeeper"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/report/zookeeper"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/local"
	_ "dubbo.apache.org/dubbo-go/v3/metadata/service/remote"
	_ "dubbo.apache.org/dubbo-go/v3/registry/nacos"
	_ "dubbo.apache.org/dubbo-go/v3/registry/zookeeper"
)

import (
	_ "github.com/apache/dubbo-kubernetes/pkg/core/reg_client/nacos"
	_ "github.com/apache/dubbo-kubernetes/pkg/core/reg_client/zookeeper"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/bootstrap/k8s"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/bootstrap/universal"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/config/k8s"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/config/universal"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/policies"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/k8s"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/resources/traditional"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/k8s"
	_ "github.com/apache/dubbo-kubernetes/pkg/plugins/runtime/universal"
)
