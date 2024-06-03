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

package server

import (
	"context"
	"net/http"
	"strconv"
)

import (
	"github.com/gin-gonic/gin"
)

import (
	ui "github.com/apache/dubbo-kubernetes/app/dubbo-ui"
	"github.com/apache/dubbo-kubernetes/pkg/config/admin"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
)

type AdminServer struct {
	Engine          *gin.Engine
	adminCfg        admin.Admin
	systemNamespace string
}

func NewAdminServer(adminCfg admin.Admin, ns string) *AdminServer {
	return &AdminServer{
		adminCfg:        adminCfg,
		systemNamespace: ns,
	}
}

func (a *AdminServer) InitHTTPRouter(rt core_runtime.Runtime) *AdminServer {
	r := gin.Default()
	// Admin UI
	r.StaticFS("/admin", http.FS(ui.FS()))
	initRouter(r, rt)
	a.Engine = r
	return a
}

func (a *AdminServer) Start(stop <-chan struct{}) error {
	errChan := make(chan error)

	var httpServer *http.Server
	httpServer = a.startHttpServer(errChan)
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

func (a *AdminServer) startHttpServer(errChan chan error) *http.Server {
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(a.adminCfg.Port),
		Handler: a.Engine,
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

func (a *AdminServer) NeedLeaderElection() bool {
	return false
}
