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

package main

import (
	"dubbo-admin-ai/config"
	"dubbo-admin-ai/manager"
	"dubbo-admin-ai/plugins/dashscope"
	"dubbo-admin-ai/utils"
)

func main() {
	mdDir := "./reference/k8s_docs/concepts"
	chunks, err := utils.ProcessMarkdownDirectory(mdDir)
	if err != nil {
		panic(err)
	}
	g := manager.Registry(dashscope.Qwen3.Key(), config.PROJECT_ROOT+"/.env", manager.ProductionLogger())
	err = utils.IndexInPinecone(g, "kube-docs", "concepts", dashscope.Qwen3_embedding.Key(), nil, chunks)
	if err != nil {
		panic(err)
	}
}
