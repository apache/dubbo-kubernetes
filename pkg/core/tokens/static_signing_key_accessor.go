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

	"github.com/pkg/errors"

	util_rsa "github.com/apache/dubbo-kubernetes/pkg/util/rsa"
)

type staticSigningKeyAccessor struct {
	keys map[string]*rsa.PublicKey
}

var _ SigningKeyAccessor = &staticSigningKeyAccessor{}

func NewStaticSigningKeyAccessor(keys []PublicKey) (SigningKeyAccessor, error) {
	s := &staticSigningKeyAccessor{
		keys: map[string]*rsa.PublicKey{},
	}
	for _, key := range keys {
		publicKey, err := util_rsa.FromPEMBytesToPublicKey([]byte(key.PEM))
		if err != nil {
			return nil, err
		}
		s.keys[key.KID] = publicKey
	}
	return s, nil
}

func (s *staticSigningKeyAccessor) GetPublicKey(ctx context.Context, keyID KeyID) (*rsa.PublicKey, error) {
	key, ok := s.keys[keyID]
	if !ok {
		return nil, &SigningKeyNotFound{
			KeyID: keyID,
		}
	}
	return key, nil
}

func (s *staticSigningKeyAccessor) GetLegacyKey(context.Context, KeyID) ([]byte, error) {
	return nil, errors.New("not supported")
}
