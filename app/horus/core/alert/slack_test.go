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

package alert

import (
	"github.com/apache/dubbo-kubernetes/app/horus/basic/config"
	"testing"
)

func TestSlackSend(t *testing.T) {
	im := &config.SlackConfiguration{
		WebhookUrl: "https://oapi.dingtalk.com/robot/send?access_token=aa2f3f74d7a2504653ca89b7a673707ba1d04b6d9d320c3572e5464d2f81471x",
	}
	SlackSend(im, "接入成功")
}
