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

package grpc_handlers

import (
	"context"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/controllers"
	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
)

type TagServiceHandler struct {
	registryv1alpha1.UnimplementedRepositoryTagServiceServer

	tagController *controllers.TagController
}

func NewTagServiceHandler() *TagServiceHandler {
	return &TagServiceHandler{
		tagController: controllers.NewTagController(),
	}
}

func (handler *TagServiceHandler) CreateRepositoryTag(ctx context.Context, req *registryv1alpha1.CreateRepositoryTagRequest) (*registryv1alpha1.CreateRepositoryTagResponse, error) {
	resp, err := handler.tagController.CreateRepositoryTag(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *TagServiceHandler) ListRepositoryTags(ctx context.Context, req *registryv1alpha1.ListRepositoryTagsRequest) (*registryv1alpha1.ListRepositoryTagsResponse, error) {
	resp, err := handler.tagController.ListRepositoryTags(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *TagServiceHandler) ListRepositoryTagsForReference(ctx context.Context, req *registryv1alpha1.ListRepositoryTagsForReferenceRequest) (*registryv1alpha1.ListRepositoryTagsForReferenceResponse, error) {
	panic("implement me")
}
