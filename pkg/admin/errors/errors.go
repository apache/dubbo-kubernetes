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
	"github.com/apache/dubbo-kubernetes/pkg/admin/model/resp"
	"github.com/pkg/errors"
)

type BizError struct {
	Code resp.Code
	Err  error
}

func (e *BizError) Error() string {
	return e.Err.Error()
}

func NewBizError(code resp.Code, err error) *BizError {
	return &BizError{
		Code: code,
		Err:  err,
	}
}

func NewBizErrorWithStack(code resp.Code, err error) *BizError {
	return &BizError{
		Code: code,
		Err:  errors.WithStack(err),
	}
}
