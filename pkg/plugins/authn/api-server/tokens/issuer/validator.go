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
	"context"

	"github.com/apache/dubbo-kubernetes/pkg/core/tokens"
	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type UserTokenValidator interface {
	Validate(ctx context.Context, token tokens.Token) (user.User, error)
}

func NewUserTokenValidator(validator tokens.Validator) UserTokenValidator {
	return &jwtTokenValidator{
		validator: validator,
	}
}

type jwtTokenValidator struct {
	validator tokens.Validator
}

func (j *jwtTokenValidator) Validate(ctx context.Context, rawToken tokens.Token) (user.User, error) {
	claims := &UserClaims{}
	if err := j.validator.ParseWithValidation(ctx, rawToken, claims); err != nil {
		return user.User{}, err
	}
	return claims.User, nil
}
