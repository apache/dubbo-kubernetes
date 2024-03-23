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

type CommitServiceHandler struct {
	registryv1alpha1.UnimplementedRepositoryCommitServiceServer

	commitController *controllers.CommitController
}

func NewCommitServiceHandler() *CommitServiceHandler {
	return &CommitServiceHandler{
		commitController: controllers.NewCommitController(),
	}
}

func (handler *CommitServiceHandler) GetRepositoryCommitByReference(ctx context.Context, req *registryv1alpha1.GetRepositoryCommitByReferenceRequest) (*registryv1alpha1.GetRepositoryCommitByReferenceResponse, error) {
	resp, err := handler.commitController.GetRepositoryCommitByReference(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *CommitServiceHandler) ListRepositoryDraftCommits(ctx context.Context, req *registryv1alpha1.ListRepositoryDraftCommitsRequest) (*registryv1alpha1.ListRepositoryDraftCommitsResponse, error) {
	resp, err := handler.commitController.ListRepositoryDraftCommits(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *CommitServiceHandler) DeleteRepositoryDraftCommit(ctx context.Context, req *registryv1alpha1.DeleteRepositoryDraftCommitRequest) (*registryv1alpha1.DeleteRepositoryDraftCommitResponse, error) {
	resp, err := handler.commitController.DeleteRepositoryDraftCommit(ctx, req)
	if err != nil {
		return nil, err.Err()
	}

	return resp, nil
}

func (handler *CommitServiceHandler) ListRepositoryCommitsByBranch(ctx context.Context, req *registryv1alpha1.ListRepositoryCommitsByBranchRequest) (*registryv1alpha1.ListRepositoryCommitsByBranchResponse, error) {
	// TODO implement me
	panic("implement me")
}

func (handler *CommitServiceHandler) GetRepositoryCommitBySequenceId(ctx context.Context, req *registryv1alpha1.GetRepositoryCommitBySequenceIdRequest) (*registryv1alpha1.GetRepositoryCommitBySequenceIdResponse, error) {
	// TODO implement me
	panic("implement me")
}
