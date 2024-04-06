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

package e

import (
	"google.golang.org/grpc/codes"
)

type UnknownError struct {
	*BaseResponseError
}

func NewUnknownError(err error) *UnknownError {
	return &UnknownError{
		NewBaseResponseError(err.Error(), codes.Unknown),
	}
}

type NotFoundError struct {
	*BaseResponseError
}

func NewNotFoundError(err error) *NotFoundError {
	return &NotFoundError{
		NewBaseResponseError(err.Error(), codes.NotFound),
	}
}

type AlreadyExistsError struct {
	*BaseResponseError
}

func NewAlreadyExistsError(err error) *AlreadyExistsError {
	return &AlreadyExistsError{
		NewBaseResponseError(err.Error(), codes.AlreadyExists),
	}
}

type PermissionDeniedError struct {
	*BaseResponseError
}

func NewPermissionDeniedError(err error) *PermissionDeniedError {
	return &PermissionDeniedError{
		NewBaseResponseError(err.Error(), codes.PermissionDenied),
	}
}

type InternalError struct {
	*BaseResponseError
}

func NewInternalError(err error) *InternalError {
	return &InternalError{
		NewBaseResponseError(err.Error(), codes.Internal),
	}
}

type CodeUnauthenticatedError struct {
	*BaseResponseError
}

func NewUnauthenticatedError(err error) *CodeUnauthenticatedError {
	return &CodeUnauthenticatedError{
		NewBaseResponseError(err.Error(), codes.Unauthenticated),
	}
}

type InvalidArgumentError struct {
	*BaseResponseError
}

func NewInvalidArgumentError(err error) *InvalidArgumentError {
	return &InvalidArgumentError{
		NewBaseResponseError(err.Error(), codes.InvalidArgument),
	}
}
