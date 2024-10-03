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

package alerter_test

import (
	"github.com/apache/dubbo-kubernetes/app/horus/base/config"
	"github.com/apache/dubbo-kubernetes/app/horus/core/alerter"
	"testing"
)

func TestSlackSend(t *testing.T) {
	im := &config.SlackConfiguration{
		WebhookUrl: "https://hooks.slack.com/services/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",
	}
	alerter.SlackSend(im, "test")
}
