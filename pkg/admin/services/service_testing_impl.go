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

package services

import (
	"context"
	"fmt"

	"github.com/jhump/protoreflect/desc"

	"github.com/apache/dubbo-kubernetes/pkg/admin/model"
	"github.com/apache/dubbo-kubernetes/pkg/admin/util"
)

type TestingServiceImpl struct{}

func (t *TestingServiceImpl) GetMethodsNames(ctx context.Context, target, serviceName string) ([]string, error) {
	ref := util.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return nil, err
	}
	defer ref.Close()

	// get methods
	methods, err := ref.ListMethods(serviceName)
	if err != nil {
		return nil, err
	}

	return methods, nil
}

// GetMethodDDetail get the detail  of method
func (t *TestingServiceImpl) GetMethodDescribe(ctx context.Context, target, methodName string) (*model.MethodDescribe, error) {
	ref := util.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return nil, err
	}

	// get descriptor of method
	descriptor, err := ref.Descriptor(methodName)
	if err != nil {
		return nil, err
	}

	methodDesc, ok := descriptor.(*desc.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a method", methodName)
	}

	// get input and output descriptor
	inputDesc := methodDesc.GetInputType()
	outputDesc := methodDesc.GetOutputType()
	// get input and output describe string
	inputDescString, err := ref.DescribeString(inputDesc.GetFullyQualifiedName())
	if err != nil {
		return nil, err
	}
	outputDescString, err := ref.DescribeString(outputDesc.GetFullyQualifiedName())
	if err != nil {
		return nil, err
	}

	return &model.MethodDescribe{
		InputName:      inputDesc.GetFullyQualifiedName(),
		InputDescribe:  inputDescString,
		OutputName:     outputDesc.GetFullyQualifiedName(),
		OutputDescribe: outputDescString,
	}, nil

}

// GetMessageTemplateString get the template string of message
func (t *TestingServiceImpl) GetMessageTemplateString(ctx context.Context, target, messageName string) (string, error) {
	ref := util.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return "", err
	}

	// get message template
	return ref.TemplateString(messageName)
}

// GetMessageDescribeString get the describe string (protobuf define string) of method
func (t *TestingServiceImpl) GetMessageDescribeString(ctx context.Context, target, messageName string) (string, error) {
	ref := util.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return "", err
	}

	// get messageName describe
	return ref.DescribeString(messageName)
}

// Invoke the target method, input is json string,
// and return the response, success and error.
func (t *TestingServiceImpl) Invoke(ctx context.Context, target, methodName, input string, headers map[string]string) (string, bool, error) {
	ref := util.NewRPCReflection(target)

	// dail the reflection server
	err := ref.Dail(ctx)
	if err != nil {
		return "", false, err
	}
	defer ref.Close()

	// set headers
	ref.SetRPCHeaders(headers)

	// invoke
	response, success, err := ref.Invoke(ctx, methodName, input)
	return response, success, err
}
