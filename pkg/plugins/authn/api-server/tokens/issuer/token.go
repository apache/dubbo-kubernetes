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

package issuer

import (
	"github.com/golang-jwt/jwt/v4"

	core_model "github.com/apache/dubbo-kubernetes/pkg/core/resources/model"
	"github.com/apache/dubbo-kubernetes/pkg/core/tokens"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

const UserTokenSigningKeyPrefix = "user-token-signing-key"

var UserTokenRevocationsGlobalSecretKey = core_model.ResourceKey{
	Name: "user-token-revocations",
	Mesh: core_model.NoMesh,
}

type UserClaims struct {
	user.User
	jwt.RegisteredClaims
}

var _ tokens.Claims = &UserClaims{}

func (c *UserClaims) ID() string {
	return c.RegisteredClaims.ID
}

func (c *UserClaims) SetRegisteredClaims(claims jwt.RegisteredClaims) {
	c.RegisteredClaims = claims
}
