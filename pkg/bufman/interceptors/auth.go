// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package interceptors

import (
	"context"
	"strings"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func Auth() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		tokens := md.Get(constant.AuthHeader)
		if len(tokens) < 1 {
			return handler(ctx, req)
		}

		// Get user id
		userID := i.auth(tokens[0])

		return handler(context.WithValue(ctx, constant.UserIDKey, userID), req)
	}
}

func (i *interceptor) auth(rawToken string) (userID string) {
	contents := strings.Split(rawToken, " ")
	if len(contents) != 2 {
		return ""
	}

	prefix, token := contents[0], contents[1]
	if prefix != constant.AuthPrefix {
		return ""
	}

	// 验证token
	tokenEntity, err := i.tokenMapper.FindAvailableByTokenName(token)
	if err != nil {
		return ""
	}

	return tokenEntity.UserID
}
