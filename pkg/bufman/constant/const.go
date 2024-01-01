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

package constant

/*
!! Warning
!! Warning
!! Warning
!! Warning

Constant can not be changed!!!
*/

const (
	AuthHeader    = "Authorization"
	AuthPrefix    = "Bearer"
	TokenLength   = 16
	CommitLength  = 32
	UserIDKey     = "user_id"
	DefaultBranch = "main"
)

const (
	FileSavaDir = "blobs"
)

const (
	MinUserNameLength = 1
	MaxUserNameLength = 200
	UserNamePattern   = "^[a-zA-Z][a-zA-Z0-9_-]*[a-zA-Z0-9]$"

	MinPasswordLength = 6
	MaxPasswordLength = 50
	PasswordPattern   = "[a-zA-Z0-9~!@&%#_]"

	MinRepositoryNameLength = 1
	MaxRepositoryNameLength = 200
	RepositoryNamePattern   = "^[a-zA-Z][a-zA-Z0-9_-]*[a-zA-Z0-9]$"

	MinDraftLength = 1
	MaxDraftLength = 20
	DraftPattern   = "^[a-zA-Z][a-zA-Z0-9_-]*[a-zA-Z0-9]$"

	MinTagLength = 1
	MaxTagLength = 20
	TagPattern   = "^[a-zA-Z][a-zA-Z0-9_-]*[a-zA-Z0-9]$"

	MinPluginLength   = 1
	MaxPluginLength   = 200
	PluginNamePattern = "^[a-zA-Z][a-zA-Z0-9_-]*[a-zA-Z0-9]$"

	MinPageSize = 1
	MaxPageSize = 50

	MinDockerRepoNameLength = 1
	MaxDockerRepoNameLength = 200
	DockerRepoNamePattern   = "^[a-zA-Z][a-zA-Z0-9_-]*[a-zA-Z0-9]$"

	MinQueryLength = 1
	MaxQueryLength = 200
	QueryPattern   = ".*"
)
