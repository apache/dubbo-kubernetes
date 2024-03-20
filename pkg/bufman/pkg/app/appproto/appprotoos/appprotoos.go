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

// Package appprotoos does OS-specific generation.
package appprotoos

import (
	"context"
	"io"
)

import (
	"go.uber.org/zap"

	"google.golang.org/protobuf/types/pluginpb"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/pkg/storage/storageos"
)

// ResponseWriter writes CodeGeneratorResponses to the OS filesystem.
type ResponseWriter interface {
	// Close writes all of the responses to disk. No further calls can be
	// made to the ResponseWriter after this call.
	io.Closer

	// AddResponse adds the response to the writer, switching on the file extension.
	// If there is a .jar extension, this generates a jar. If there is a .zip
	// extension, this generates a zip. If there is no extension, this outputs
	// to the directory.
	//
	// pluginOut will be unnormalized within this function.
	AddResponse(
		ctx context.Context,
		response *pluginpb.CodeGeneratorResponse,
		pluginOut string,
	) error
}

// NewResponseWriter returns a new ResponseWriter.
func NewResponseWriter(
	logger *zap.Logger,
	storageosProvider storageos.Provider,
	options ...ResponseWriterOption,
) ResponseWriter {
	return newResponseWriter(
		logger,
		storageosProvider,
		options...,
	)
}

// ResponseWriterOption is an option for the ResponseWriter.
type ResponseWriterOption func(*responseWriterOptions)

// ResponseWriterWithCreateOutDirIfNotExists returns a new ResponseWriterOption that creates
// the directory if it does not exist.
func ResponseWriterWithCreateOutDirIfNotExists() ResponseWriterOption {
	return func(responseWriterOptions *responseWriterOptions) {
		responseWriterOptions.createOutDirIfNotExists = true
	}
}
