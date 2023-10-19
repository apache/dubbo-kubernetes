//Licensed to the Apache Software Foundation (ASF) under one or more
//contributor license agreements.  See the NOTICE file distributed with
//this work for additional information regarding copyright ownership.
//The ASF licenses this file to You under the Apache License, Version 2.0
//(the "License"); you may not use this file except in compliance with
//the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package server

import (
	"context"
	"net/http"

	webhookclient "github.com/apache/dubbo-kubernetes/pkg/core/client/webhook"

	dubbo_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	cert "github.com/apache/dubbo-kubernetes/pkg/core/cert/provider"
	"github.com/apache/dubbo-kubernetes/pkg/core/logger"
	"github.com/apache/dubbo-kubernetes/pkg/webhook/patch"
	"github.com/apache/dubbo-kubernetes/pkg/webhook/webhook"
)

type WebhookServer struct {
	Options       *dubbo_cp.Config
	WebhookClient webhookclient.Client
	CertStorage   *cert.CertStorage

	WebhookServer *webhook.Webhook
	DubboInjector *patch.DubboSdk
}

func NewServer(options *dubbo_cp.Config) *WebhookServer {
	return &WebhookServer{Options: options}
}

func (s *WebhookServer) NeedLeaderElection() bool {
	return false
}

func (s *WebhookServer) Start(stop <-chan struct{}) error {
	errChan := make(chan error)
	if s.Options.KubeConfig.InPodEnv {
		go func() {
			err := s.WebhookServer.Server.ListenAndServeTLS("", "")
			if err != nil {
				switch err {
				case http.ErrServerClosed:
					logger.Sugar().Info("[Webhook] shutting down HTTP Server")
				default:
					logger.Sugar().Error(err, "[Webhook] could not start an HTTP Server")
					errChan <- err
				}
			}
		}()
		s.WebhookClient.UpdateWebhookConfig(s.Options, s.CertStorage.GetAuthorityCert().CertPem)
		select {
		case <-stop:
			logger.Sugar().Info("[Webhook] stopping Authority")
			if s.WebhookServer.Server != nil {
				return s.WebhookServer.Server.Shutdown(context.Background())
			}
		case err := <-errChan:
			return err
		}
	}
	return nil
}
