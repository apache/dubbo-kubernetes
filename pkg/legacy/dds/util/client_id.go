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

package util

import (
	"context"
	"fmt"
)

import (
	"github.com/pkg/errors"

	"google.golang.org/grpc/metadata"
)

const clientIDKey = "client-id"

// ClientIDFromIncomingCtx returns the ID of the peer. Global has the ID
// "global" while zones have the zone name. This is also known as the peer ID.
func ClientIDFromIncomingCtx(ctx context.Context) (string, error) {
	return MetadataFromIncomingCtx(ctx, clientIDKey)
}

func MetadataFromIncomingCtx(ctx context.Context, key string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("metadata is not provided")
	}
	if len(md[key]) == 0 {
		return "", errors.New(fmt.Sprintf("%q is not present in metadata", key))
	}
	return md[key][0], nil
}
