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

package errors

import (
	"fmt"
	"reflect"
)

type Unauthenticated struct{}

func (e *Unauthenticated) Error() string {
	return "Unauthenticated"
}

type MethodNotAllowed struct{}

func (e *MethodNotAllowed) Error() string {
	return "Method not allowed"
}

type Conflict struct{}

func (e *Conflict) Error() string {
	return "Conflict"
}

type ServiceUnavailable struct{}

func (e *ServiceUnavailable) Error() string {
	return "Service unavailable"
}

type BadRequest struct {
	msg string
}

func NewBadRequestError(msg string) error {
	return &BadRequest{msg: msg}
}

func (e *BadRequest) Error() string {
	if e.msg == "" {
		return "bad request"
	}
	return fmt.Sprintf("bad request: %s", e.msg)
}

func (e *BadRequest) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}

func NewNotFoundError(msg string) error {
	return &NotFound{msg: msg}
}

type NotFound struct {
	msg string
}

func (e *NotFound) Error() string {
	if e.msg == "" {
		return "not found"
	}
	return fmt.Sprintf("not found: %s", e.msg)
}

func (e *NotFound) Is(err error) bool {
	return reflect.TypeOf(e) == reflect.TypeOf(err)
}
