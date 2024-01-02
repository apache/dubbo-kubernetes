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

	registryv1alpha1 "github.com/apache/dubbo-kubernetes/pkg/bufman/gen/proto/go/registry/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SearchServiceHandler struct {
	registryv1alpha1.UnimplementedSearchServiceServer
}

func (SearchServiceHandler) SearchUser(ctx context.Context, req *registryv1alpha1.SearchUserRequest) (*registryv1alpha1.SearchUserResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchUser not implemented")
}
func (SearchServiceHandler) SearchRepository(ctx context.Context, req *registryv1alpha1.SearchRepositoryRequest) (*registryv1alpha1.SearchRepositoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchRepository not implemented")
}
func (SearchServiceHandler) SearchLastCommitByContent(ctx context.Context, req *registryv1alpha1.SearchLastCommitByContentRequest) (*registryv1alpha1.SearchLastCommitByContentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchLastCommitByContent not implemented")
}
func (SearchServiceHandler) SearchCurationPlugin(ctx context.Context, req *registryv1alpha1.SearchCuratedPluginRequest) (*registryv1alpha1.SearchCuratedPluginResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchCurationPlugin not implemented")
}
func (SearchServiceHandler) SearchTag(ctx context.Context, req *registryv1alpha1.SearchTagRequest) (*registryv1alpha1.SearchTagResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchTag not implemented")
}
func (SearchServiceHandler) SearchDraft(ctx context.Context, req *registryv1alpha1.SearchDraftRequest) (*registryv1alpha1.SearchDraftResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SearchDraft not implemented")
}
