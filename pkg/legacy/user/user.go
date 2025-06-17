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

package user

import (
	"strings"
)

const AuthenticatedGroup = "mesh-system:authenticated"

type User struct {
	Name   string
	Groups []string
}

func (u User) String() string {
	return u.Name + "/" + strings.Join(u.Groups, ",")
}

func (u User) Authenticated() User {
	u.Groups = append(u.Groups, AuthenticatedGroup)
	return u
}

// Admin is a static user that can be used when authn mechanism does not authenticate to specific user,
// but authenticate to admin without giving credential (ex. authenticate as localhost, authenticate via legacy client certs).
var Admin = User{
	Name:   "mesh-system:admin",
	Groups: []string{"mesh-system:admin"},
}

var Anonymous = User{
	Name:   "mesh-system:anonymous",
	Groups: []string{"mesh-system:unauthenticated"},
}

// ControlPlane is a static user that is used whenever the control plane itself executes operations.
// For example: update of DataplaneInsight, creation of default resources etc.
var ControlPlane = User{
	Name:   "mesh-system:control-plane",
	Groups: []string{},
}
