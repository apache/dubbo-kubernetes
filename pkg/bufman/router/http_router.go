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

package router

import (
	"context"
	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	"net/http"
	"strconv"

	"github.com/apache/dubbo-kubernetes/pkg/bufman/handlers/http_handlers"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/gin-gonic/gin"
)

type HTTPRouter struct {
	Engine *gin.Engine
	Port   int
}

func (r *HTTPRouter) Start(stop <-chan struct{}) error {
	errChan := make(chan error)

	var httpServer *http.Server
	httpServer = r.startHttpServer(errChan)
	select {
	case <-stop:
		logger.Sugar().Info("stopping bufman")
		if httpServer != nil {
			return httpServer.Shutdown(context.Background())
		}
	case err := <-errChan:
		return err
	}
	return nil
}

func (r *HTTPRouter) startHttpServer(errChan chan error) *http.Server {
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(r.Port),
		Handler: r.Engine,
	}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			switch err {
			case http.ErrServerClosed:
				logger.Sugar().Info("shutting down bufman HTTP Server")
			default:
				logger.Sugar().Error(err, "could not start bufman HTTP Server")
				errChan <- err
			}
		}
	}()
	return server
}

func (r *HTTPRouter) NeedLeaderElection() bool {
	return false
}

func InitHTTPRouter(config dubbo_cp.Config) *HTTPRouter {
	r := gin.Default()

	router := r.Group("/api/v1")
	authn := router.Group("/authn")
	{
		authn.GET("/current_user", http_handlers.AuthnGroup.GetCurrentUser) // 根据token获取当前用户信息
	}

	user := router.Group("/user")
	{
		user.POST("/create", http_handlers.UserGroup.CreateUser) // 创建用户
		user.GET("/:id", http_handlers.UserGroup.GetUser)        // 查询用户
		user.POST("/list", http_handlers.UserGroup.ListUsers)    // 批量查询用户
	}

	token := router.Group("/token")
	{
		token.POST("/create", http_handlers.TokenGroup.CreateToken)      // 创建token
		token.GET("/:token_id", http_handlers.TokenGroup.GetToken)       // 获取token
		token.POST("/list", http_handlers.TokenGroup.ListTokens)         // 批量查询token
		token.DELETE("/:token_id", http_handlers.TokenGroup.DeleteToken) // 删除tokens
	}

	repository := router.Group("/repository")
	{
		repository.POST("/create", http_handlers.RepositoryGroup.CreateRepositoryByFullName)             // 创建repository
		repository.GET("/:id", http_handlers.RepositoryGroup.GetRepository)                              // 根据id获取repository
		repository.POST("/list", http_handlers.RepositoryGroup.ListRepositories)                         // 批量查询所有repository
		repository.DELETE("/:id", http_handlers.RepositoryGroup.DeleteRepository)                        // 删除repository
		repository.POST("/list/:user_id", http_handlers.RepositoryGroup.ListUserRepositories)            // 批量查询用户的repository
		repository.POST("/list_accessible", http_handlers.RepositoryGroup.ListRepositoriesUserCanAccess) // 批量查询当前用户可访问的repository
		repository.PUT("/deprecate", http_handlers.RepositoryGroup.DeprecateRepositoryByName)            // 弃用repository
		repository.PUT("/undeprecate", http_handlers.RepositoryGroup.UndeprecateRepositoryByName)        // 解除弃用
		repository.PUT("/update", http_handlers.RepositoryGroup.UpdateRepositorySettingsByName)          // 更新repository

		commit := repository.Group("/commit")
		{
			commit.POST("/list/:repository_owner/:repository_name/:reference", http_handlers.CommitGroup.ListRepositoryCommitsByReference) // 获取reference对应commit以及之前的commits
			commit.GET("/:repository_owner/:repository_name/:reference", http_handlers.CommitGroup.GetRepositoryCommitByReference)         // 获取reference对应commit
			commit.POST("/draft/list/:repository_owner/:repository_name", http_handlers.CommitGroup.ListRepositoryDraftCommits)            // 获取所有的草稿
			commit.DELETE("/draft/:repository_owner/:repository_name/:draft_name", http_handlers.CommitGroup.DeleteRepositoryDraftCommit)  // 删除草稿
		}

		tag := repository.Group("/tag")
		{
			tag.POST("/create", http_handlers.TagGroup.CreateRepositoryTag) // 创建tag
			tag.POST("/list", http_handlers.TagGroup.ListRepositoryTags)    // 查询repository下的所有tag
		}

		doc := repository.Group("/doc")
		{
			doc.GET("/source/:repository_owner/:repository_name/:reference", http_handlers.DocGroup.GetSourceDirectoryInfo)                 // 获取目录信息
			doc.GET("/source/:repository_owner/:repository_name/:reference/:path", http_handlers.DocGroup.GetSourceFile)                    // 获取文件源码
			doc.GET("/module/:repository_owner/:repository_name/:reference", http_handlers.DocGroup.GetModuleDocumentation)                 // 获取repo说明文档
			doc.GET("/package/:repository_owner/:repository_name/:reference", http_handlers.DocGroup.GetModulePackages)                     // 获取repo packages
			doc.GET("/package/:repository_owner/:repository_name/:reference/:package_name", http_handlers.DocGroup.GetPackageDocumentation) // 获取包说明文档
		}

		search := router.Group("/search")
		{
			search.POST("/user", http_handlers.SearchGroup.SearchUser)                  // 搜索用户
			search.POST("/repository", http_handlers.SearchGroup.SearchRepository)      // 搜索仓库
			search.POST("/commit", http_handlers.SearchGroup.SearchLastCommitByContent) // 搜索根据内容搜索最近一次提交
			search.POST("/tag", http_handlers.SearchGroup.SearchTag)                    // 搜索tag
			search.POST("/draft", http_handlers.SearchGroup.SearchDraft)                // 搜索草稿
		}
	}

	return &HTTPRouter{
		Engine: r,
		Port:   config.Bufman.Server.HTTPPort,
	}
}
