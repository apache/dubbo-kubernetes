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

package security

import (
	"errors"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/config"
	"github.com/golang-jwt/jwt/v4"
)

type PageTokenChaim struct {
	PageOffset int
	jwt.RegisteredClaims
}

func GenerateNextPageToken(lastPageOffset, lastPageSize, lastDataLength int) (string, error) {
	if lastDataLength < lastPageSize {
		// 已经查询完了
		return "", nil
	}

	nextPageOffset := lastPageOffset + lastDataLength
	// 定义 token 的过期时间
	now := time.Now()
	expireTime := now.Add(config.Properties.Server.PageTokenExpireTime)

	// 创建一个自定义的 Claim
	chaim := &PageTokenChaim{
		PageOffset: nextPageOffset,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "bufman",
		},
	}

	// 使用 JWT 签名算法生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, chaim)

	// 将 token 进行加盐加密
	tokenString, err := token.SignedString([]byte(config.Properties.Server.PageTokenSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParsePageToken(tokenString string) (*PageTokenChaim, error) {
	if tokenString == "" {
		return &PageTokenChaim{
			PageOffset: 0,
		}, nil
	}

	// 解析 token
	token, err := jwt.ParseWithClaims(tokenString, &PageTokenChaim{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.Properties.Server.PageTokenSecret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*PageTokenChaim); ok && token.Valid {
		return claims, nil
	} else {
		return nil, errors.New("invalid page token")
	}
}
