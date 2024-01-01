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
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/constant"
)

// EncryptPlainPassword 加密明文密码
func EncryptPlainPassword(userName, plainPwd string) string {
	sha := sha256.New()
	sha.Write([]byte(plainPwd))
	sha.Write([]byte(userName))
	bytes := sha.Sum(nil)

	return hex.EncodeToString(bytes)
}

// GenerateToken 生成token
func GenerateToken(username, note string) string {
	sha := sha256.New()
	// 以用户名 note 时间戳做哈希
	sha.Write([]byte(username))
	sha.Write([]byte(note))
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	sha.Write([]byte(now))

	return hex.EncodeToString(sha.Sum(nil)[:constant.TokenLength/2])
}

func GenerateCommitName(userName, RepositoryName string) string {
	sha := sha256.New()
	// 以用户名 note 时间戳做哈希
	sha.Write([]byte(userName))
	sha.Write([]byte(RepositoryName))
	now := strconv.FormatInt(time.Now().UnixNano(), 10)
	sha.Write([]byte(now))
	bytes := sha.Sum(nil)

	return hex.EncodeToString(bytes[:constant.CommitLength/2])
}
