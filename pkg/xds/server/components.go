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
	"github.com/pkg/errors"
)

import (
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/apache/dubbo-kubernetes/pkg/xds/cache/cla"
	xds_context "github.com/apache/dubbo-kubernetes/pkg/xds/context"
	v3 "github.com/apache/dubbo-kubernetes/pkg/xds/server/v3"
)

func RegisterXDS(rt core_runtime.Runtime) error {
	claCache, err := cla.NewCache(rt.Config().Store.Cache.ExpirationTime.Duration)
	if err != nil {
		return err
	}

	envoyCpCtx := &xds_context.ControlPlaneContext{
		CLACache: claCache,
		Zone:     "",
	}
	if err := v3.RegisterXDS(nil, envoyCpCtx, rt); err != nil {
		return errors.Wrap(err, "could not register V3 XDS")
	}
	return nil
}
