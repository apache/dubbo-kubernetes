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

package tokens_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kuma_cp "github.com/apache/dubbo-kubernetes/pkg/config/app/dubbo-cp"
	store_config "github.com/apache/dubbo-kubernetes/pkg/config/core/resources/store"
	"github.com/apache/dubbo-kubernetes/pkg/core"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/apis/system"
	"github.com/apache/dubbo-kubernetes/pkg/core/resources/manager"
	core_store "github.com/apache/dubbo-kubernetes/pkg/core/resources/store"
	core_tokens "github.com/apache/dubbo-kubernetes/pkg/core/tokens"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/authn/api-server/tokens/issuer"
	"github.com/apache/dubbo-kubernetes/pkg/plugins/resources/memory"
)

var _ = Describe("Admin Token Bootstrap", func() {
	It("should bootstrap admin token", func() {
		// given
		ctx := context.Background()
		resManager := manager.NewResourceManager(memory.NewStore())
		signingKeyManager := core_tokens.NewSigningKeyManager(resManager, issuer.UserTokenSigningKeyPrefix)
		tokenIssuer := issuer.NewUserTokenIssuer(core_tokens.NewTokenIssuer(signingKeyManager))
		tokenValidator := issuer.NewUserTokenValidator(
			core_tokens.NewValidator(
				core.Log.WithName("test"),
				[]core_tokens.SigningKeyAccessor{
					core_tokens.NewSigningKeyAccessor(resManager, issuer.UserTokenSigningKeyPrefix),
				},
				core_tokens.NewRevocations(resManager, issuer.UserTokenRevocationsGlobalSecretKey),
				store_config.MemoryStore,
			),
		)

		component := tokens.NewAdminTokenBootstrap(tokenIssuer, resManager, kuma_cp.DefaultConfig())
		err := signingKeyManager.CreateDefaultSigningKey(ctx)
		Expect(err).ToNot(HaveOccurred())
		stopCh := make(chan struct{})
		defer close(stopCh)

		// when
		go func() {
			_ = component.Start(stopCh) // it never returns an error
		}()

		// then token is created
		Eventually(func(g Gomega) {
			globalSecret := system.NewGlobalSecretResource()
			err = resManager.Get(context.Background(), globalSecret, core_store.GetBy(tokens.AdminTokenKey))
			g.Expect(err).ToNot(HaveOccurred())
			user, err := tokenValidator.Validate(ctx, string(globalSecret.Spec.Data.Value))
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(user.Name).To(Equal("mesh-system:admin"))
			g.Expect(user.Groups).To(Equal([]string{"mesh-system:admin"}))
		}).Should(Succeed())
	})
})
