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

package search

import (
	"context"
	"sync"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/model"
)

type Searcher interface {
	SearchUsers(ctx context.Context, query string, offset, limit int, reverse bool) (model.Users, error)
	SearchRepositories(ctx context.Context, query string, offset, limit int, reverse bool) (model.Repositories, error)
	SearchCommitsByContent(ctx context.Context, userID string, query string, offset, limit int, reverse bool) (model.Commits, error)
	SearchTag(ctx context.Context, repositoryID string, query string, offset, limit int, reverse bool) (model.Tags, error)
	SearchDraft(ctx context.Context, repositoryID string, query string, offset, limit int, reverse bool) (model.Commits, error)
}

// 单例模式
var (
	searcher Searcher
	once     sync.Once
)

func NewSearcher() Searcher {
	if searcher == nil {
		// 对象初始化
		once.Do(func() {
			searcher = NewDBSearcher()
		})
	}

	return searcher
}
