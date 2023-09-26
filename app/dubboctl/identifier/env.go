// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package identifier

import (
	"net/url"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/filesystem"
	"github.com/apache/dubbo-kubernetes/deploy"
)

var (
	// deploy dir is root in embed.FS
	deployUri = &url.URL{
		Scheme:   filesystem.EmbedSchema,
		OmitHost: true,
	}

	chartsUri          = deployUri.JoinPath("charts")
	profilesUri        = deployUri.JoinPath("profiles")
	addonsUri          = deployUri.JoinPath("addons")
	addonDashboardsUri = addonsUri.JoinPath("dashboards")
	addonManifestsUri  = addonsUri.JoinPath("manifests")

	Charts          = chartsUri.String()
	Profiles        = profilesUri.String()
	Addons          = addonsUri.String()
	AddonDashboards = addonDashboardsUri.String()
	AddonManifests  = addonManifestsUri.String()
)

var (
	UnionFS filesystem.UnionFS
)

func init() {
	UnionFS = filesystem.NewUnionFS(deploy.EmbedRootFS)
}
