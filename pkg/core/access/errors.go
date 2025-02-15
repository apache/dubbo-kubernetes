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

package access

import (
	"fmt"
	"reflect"

	"github.com/apache/dubbo-kubernetes/pkg/core/user"
)

type AccessDeniedError struct {
	Reason string
}

func (a *AccessDeniedError) Error() string {
	return "access denied: " + a.Reason
}

func (a *AccessDeniedError) Is(err error) bool {
	return reflect.TypeOf(a) == reflect.TypeOf(err)
}

func Validate(usernames map[string]struct{}, groups map[string]struct{}, user user.User, action string) error {
	if _, ok := usernames[user.Name]; ok {
		return nil
	}
	for _, group := range user.Groups {
		if _, ok := groups[group]; ok {
			return nil
		}
	}
	return &AccessDeniedError{Reason: fmt.Sprintf("user %q cannot access %s", user, action)}
}
