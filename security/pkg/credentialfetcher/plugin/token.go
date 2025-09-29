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


package plugin

import (
	"k8s.io/klog/v2"
	"os"
	"strings"
)

type KubernetesTokenPlugin struct {
	path string
}

func CreateTokenPlugin(path string) *KubernetesTokenPlugin {
	return &KubernetesTokenPlugin{
		path: path,
	}
}

func (t KubernetesTokenPlugin) GetPlatformCredential() (string, error) {
	if t.path == "" {
		return "", nil
	}
	tok, err := os.ReadFile(t.path)
	if err != nil {
		klog.Warningf("failed to fetch token from file: %v", err)
		return "", nil
	}
	return strings.TrimSpace(string(tok)), nil
}

func (t KubernetesTokenPlugin) GetIdentityProvider() string {
	return ""
}

func (t KubernetesTokenPlugin) Stop() {
}
