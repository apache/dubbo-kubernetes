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
	"crypto/x509"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	util_rsa "github.com/apache/dubbo-kubernetes/pkg/util/rsa"
)

func NewSigningKey() ([]byte, error) {
	key, err := util_rsa.GenerateKey(util_rsa.DefaultKeySize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate RSA key")
	}
	return util_rsa.FromPrivateKeyToPEMBytes(key)
}

func SigningKeyResourceKey(signingKeyPrefix string, keyID KeyID, mesh string) model.ResourceKey {
	name := fmt.Sprintf("%s-%s", signingKeyPrefix, keyID)
	if keyID == KeyIDFallbackValue { // backwards compatibility with 1.3.x signing keys https://github.com/kumahq/kuma/issues/4006
		name = signingKeyPrefix
	}
	return model.ResourceKey{
		Name: name,
		Mesh: mesh,
	}
}

type SigningKeyNotFound struct {
	KeyID  KeyID
	Prefix string
	Mesh   string
}

func (s *SigningKeyNotFound) Error() string {
	key := SigningKeyResourceKey(s.Prefix, s.KeyID, s.Mesh)
	if s.Mesh == "" {
		return fmt.Sprintf("there is no signing key with KID %s. GlobalSecret of name %q is not found. If signing key was rotated, regenerate the token", s.KeyID, key.Name)
	} else {
		return fmt.Sprintf("there is no signing key with KID %s. Secret of name %q in mesh %q is not found. If signing key was rotated, regenerate the token", s.KeyID, key.Name, key.Mesh)
	}
}

func IsSigningKeyNotFound(err error) bool {
	target := &SigningKeyNotFound{}
	return errors.As(err, &target)
}

func keyBytesToRsaPrivateKey(keyBytes []byte) (*rsa.PrivateKey, error) {
	if util_rsa.IsPrivateKeyPEMBytes(keyBytes) {
		key, err := util_rsa.FromPEMBytesToPrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	}

	// support non-PEM RSA key for legacy reasons
	key, err := x509.ParsePKCS1PrivateKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func keyBytesToRsaPublicKey(keyBytes []byte) (*rsa.PublicKey, error) {
	if util_rsa.IsPublicKeyPEMBytes(keyBytes) {
		key, err := util_rsa.FromPEMBytesToPublicKey(keyBytes)
		if err != nil {
			return nil, err
		}
		return key, nil
	}

	// support non-PEM RSA key for legacy reasons
	key, err := x509.ParsePKCS1PublicKey(keyBytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func signingKeySerialNumber(secretName string, signingKeyPrefix string) (int, error) {
	serialNumberStr := strings.ReplaceAll(secretName, signingKeyPrefix+"-", "")
	serialNumber, err := strconv.Atoi(serialNumberStr)
	if err != nil {
		return 0, err
	}
	return serialNumber, nil
}

func getKeyBytes(
	ctx context.Context,
	resManager manager.ReadOnlyResourceManager,
	signingKeyPrefix string,
	keyID KeyID,
) ([]byte, error) {
	resource := system.NewGlobalSecretResource()
	if err := resManager.Get(ctx, resource, store.GetBy(SigningKeyResourceKey(signingKeyPrefix, keyID, model.NoMesh))); err != nil {
		if store.IsResourceNotFound(err) {
			return nil, &SigningKeyNotFound{
				KeyID:  keyID,
				Prefix: signingKeyPrefix,
			}
		}

		return nil, errors.Wrap(err, "could not retrieve signing key")
	}

	return resource.Spec.GetData().GetValue(), nil
}
