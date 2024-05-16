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

package bufman

import (
	"github.com/apache/dubbo-kubernetes/pkg/bufman/router"
	core_runtime "github.com/apache/dubbo-kubernetes/pkg/core/runtime"
	"github.com/pkg/errors"
)

func Setup(rt core_runtime.Runtime) error {
	if !rt.Config().Bufman.OpenBufman {
		// dont open bufman
		return nil
	}

	if err := InitConfig(rt); err != nil {
		return errors.Wrap(err, "Bufman init config failed")
	}

	if err := RegisterDatabase(rt); err != nil {
		return errors.Wrap(err, "Bufman Database register failed")
	}

	httpRouter := router.InitHTTPRouter(rt.Config())
	grpcRouter := router.InitGRPCRouter(rt.Config())

	if err := rt.Add(httpRouter); err != nil {
		return errors.Wrap(err, "Add Bufman HTTP Server Component failed")
	}
	if err := rt.Add(grpcRouter); err != nil {
		return errors.Wrap(err, "Add Bufman GRPC Server Component failed")
	}
	return nil
}
