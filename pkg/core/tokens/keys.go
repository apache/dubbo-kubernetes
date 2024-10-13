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

package tokens

import (
	"os"

	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

type PublicKey struct {
	PEM string
	KID string
}

func PublicKeyFromConfig(publicKeys []config_types.PublicKey) ([]PublicKey, error) {
	var keys []PublicKey
	for _, key := range publicKeys {
		publicKey, err := configKeyToCoreKey(key)
		if err != nil {
			return nil, err
		}
		keys = append(keys, publicKey)
	}
	return keys, nil
}

func PublicKeyByMeshFromConfig(publicKeys []config_types.MeshedPublicKey) (map[string][]PublicKey, error) {
	byMesh := map[string][]PublicKey{}
	for _, key := range publicKeys {
		keys, ok := byMesh[key.Mesh]
		if !ok {
			keys = []PublicKey{}
		}
		publicKey, err := configKeyToCoreKey(key.PublicKey)
		if err != nil {
			return nil, err
		}
		keys = append(keys, publicKey)
		byMesh[key.Mesh] = keys
	}
	return byMesh, nil
}

func configKeyToCoreKey(key config_types.PublicKey) (PublicKey, error) {
	publicKey := PublicKey{
		KID: key.KID,
	}
	if key.KeyFile != "" {
		content, err := os.ReadFile(key.KeyFile)
		if err != nil {
			return PublicKey{}, err
		}
		publicKey.PEM = string(content)
	}
	if key.Key != "" {
		publicKey.PEM = key.Key
	}
	return publicKey, nil
}
