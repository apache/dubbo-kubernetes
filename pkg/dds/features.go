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

package dds

import (
	"context"
)

import (
	"golang.org/x/exp/slices"

	"google.golang.org/grpc/metadata"
)

// Features is a set of available features for the control plane.
// If by any chance we get into a situation that we need to execute a logic conditionally on capabilities of control plane,
// instead of defining conditions on version which is fragile, we can define a condition based on features.
type Features map[string]bool

func (f Features) HasFeature(feature string) bool {
	return f[feature]
}

const FeaturesMetadataKey string = "features"

// FeatureZoneToken means that the zone control plane can handle incoming Zone Token from global control plane.
const FeatureZoneToken string = "zone-token"

// FeatureZonePingHealth means that the zone control plane sends pings to the
// global control plane to indicate it's still running.
const FeatureZonePingHealth string = "zone-ping-health"

// FeatureHashSuffix means that the zone control plane has a fix for the MeshGateway renaming
// issue https://github.com/kumahq/kuma/pull/8450 and can handle the hash suffix in the resource name.
const FeatureHashSuffix string = "hash-suffix"

func ContextHasFeature(ctx context.Context, feature string) bool {
	md, _ := metadata.FromIncomingContext(ctx)
	features := md.Get(FeaturesMetadataKey)
	return slices.Contains(features, feature)
}
