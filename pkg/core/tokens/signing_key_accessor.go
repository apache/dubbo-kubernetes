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
	"context"
	"crypto/rsa"

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
)

// SigningKeyAccessor access public part of signing key
// In the future, we may add offline token generation (kumactl without CP running or external system)
// In that case, we could provide only public key to the CP via static configuration.
// So we can easily do this by providing separate implementation for this interface.
type SigningKeyAccessor interface {
	GetPublicKey(context.Context, KeyID) (*rsa.PublicKey, error)
	// GetLegacyKey returns legacy key. In pre 1.4.x version of Kuma, we used symmetric HMAC256 method of signing DP keys.
	// In that case, we have to retrieve private key even for verification.
	GetLegacyKey(context.Context, KeyID) ([]byte, error)
}

type signingKeyAccessor struct {
	resManager       manager.ReadOnlyResourceManager
	signingKeyPrefix string
}

var _ SigningKeyAccessor = &signingKeyAccessor{}

func NewSigningKeyAccessor(resManager manager.ReadOnlyResourceManager, signingKeyPrefix string) SigningKeyAccessor {
	return &signingKeyAccessor{
		resManager:       resManager,
		signingKeyPrefix: signingKeyPrefix,
	}
}

func (s *signingKeyAccessor) GetPublicKey(ctx context.Context, keyID KeyID) (*rsa.PublicKey, error) {
	keyBytes, err := getKeyBytes(ctx, s.resManager, s.signingKeyPrefix, keyID)
	if err != nil {
		return nil, err
	}

	key, err := keyBytesToRsaPrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return &key.PublicKey, nil
}

func (s *signingKeyAccessor) GetLegacyKey(ctx context.Context, keyID KeyID) ([]byte, error) {
	return getKeyBytes(ctx, s.resManager, s.signingKeyPrefix, keyID)
}
