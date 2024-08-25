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
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	system_proto "github.com/apache/dubbo-kubernetes/api/system/v1alpha1"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
)

const (
	DefaultKeyID = "1"
)

// SigningKeyManager manages tokens's signing keys.
// We can have many signing keys in the system.
// Example: "user-token-signing-key-1", "user-token-signing-key-2" etc.
// "user-token-signing-key" has a serial number of 0
// The latest key is  a key with a higher serial number (number at the end of the name)
type SigningKeyManager interface {
	GetLatestSigningKey(context.Context) (*rsa.PrivateKey, string, error)
	CreateDefaultSigningKey(context.Context) error
	CreateSigningKey(ctx context.Context, keyID KeyID) error
}

func NewSigningKeyManager(manager manager.ResourceManager, signingKeyPrefix string) SigningKeyManager {
	return &signingKeyManager{
		manager:          manager,
		signingKeyPrefix: signingKeyPrefix,
	}
}

type signingKeyManager struct {
	manager          manager.ResourceManager
	signingKeyPrefix string
}

var _ SigningKeyManager = &signingKeyManager{}

func (s *signingKeyManager) GetLatestSigningKey(ctx context.Context) (*rsa.PrivateKey, string, error) {
	resources := system.GlobalSecretResourceList{}
	if err := s.manager.List(ctx, &resources); err != nil {
		return nil, "", errors.Wrap(err, "could not retrieve signing key from secret manager")
	}
	return latestSigningKey(&resources, s.signingKeyPrefix, model.NoMesh)
}

func latestSigningKey(list model.ResourceList, prefix string, mesh string) (*rsa.PrivateKey, string, error) {
	var signingKey model.Resource
	highestSerialNumber := -1
	for _, resource := range list.GetItems() {
		if !strings.HasPrefix(resource.GetMeta().GetName(), prefix) {
			continue
		}
		serialNumber, _ := signingKeySerialNumber(resource.GetMeta().GetName(), prefix)
		if serialNumber > highestSerialNumber {
			signingKey = resource
			highestSerialNumber = serialNumber
		}
	}

	if signingKey == nil {
		return nil, "", &SigningKeyNotFound{
			KeyID:  DefaultKeyID,
			Prefix: prefix,
			Mesh:   mesh,
		}
	}

	key, err := keyBytesToRsaPrivateKey(signingKey.GetSpec().(*system_proto.Secret).GetData().GetValue())
	if err != nil {
		return nil, "", err
	}

	return key, strconv.Itoa(highestSerialNumber), nil
}

func (s *signingKeyManager) CreateDefaultSigningKey(ctx context.Context) error {
	return s.CreateSigningKey(ctx, DefaultKeyID)
}

func (s *signingKeyManager) CreateSigningKey(ctx context.Context, keyID KeyID) error {
	key, err := NewSigningKey()
	if err != nil {
		return err
	}

	secret := system.NewGlobalSecretResource()
	secret.Spec = &system_proto.Secret{
		Data: &wrapperspb.BytesValue{
			Value: key,
		},
	}
	return s.manager.Create(ctx, secret, store.CreateBy(SigningKeyResourceKey(s.signingKeyPrefix, keyID, model.NoMesh)))
}
