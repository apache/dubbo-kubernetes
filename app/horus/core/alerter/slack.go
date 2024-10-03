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

package alerter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/apache/dubbo-kubernetes/app/horus/base/config"
	"k8s.io/klog/v2"
	"net/http"
)

const SlackTitle = "项目组"

type Text struct {
	Text string `json:"text"`
}

func SlackSend(sk *config.SlackConfiguration, channel string) {
	skm := Text{Text: "text"}
	skm.Text = fmt.Sprintf("%s"+
		"%v", SlackTitle, channel)
	bs, err := json.Marshal(skm)
	if err != nil {
		klog.Errorf("slack json marshal err:%v\n dtm:%v\n", err, skm)
	}
	res, err := http.Post(sk.WebhookUrl, "application/json", bytes.NewBuffer(bs))
	if res.StatusCode != 200 {
		klog.Errorf("send slack status code err:%v\n code:%v\n channel:%v\n", err, res.StatusCode, channel)
		return
	}
}
