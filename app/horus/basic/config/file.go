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

package config

type Config struct {
	Address        string                 `yaml:"address"`
	KubeTimeSecond int64                  `yaml:"kubeTimeSecond"`
	Mysql          *MysqlConfiguration    `yaml:"mysql"`
	DingTalk       *DingTalkConfiguration `yaml:"dingTalk"`
	Slack          *SlackConfiguration    `yaml:"slack"`
	KubeMultiple   map[string]string      `yaml:"kubeMultiple"`
	PromMultiple   map[string]string      `yaml:"promMultiple"`
	NodeRecovery   *RecoveryConfiguration `yaml:"nodeRecovery"`
	CustomModular  *ModularConfiguration  `yaml:"customModular"`
	NodeDownTime   *DowntimeConfiguration `yaml:"nodeDownTime"`
	PodAbnormal    *AbnormalConfiguration `yaml:"podAbnormal"`
}

type MysqlConfiguration struct {
	Name  string `yaml:"name"`
	Addr  string `yaml:"addr"`
	Debug bool   `yaml:"debug"`
}

type DingTalkConfiguration struct {
	WebhookUrl string   `yaml:"webhookUrl"`
	Title      string   `yaml:"title"`
	AtMobiles  []string `yaml:"atMobiles"`
}

type SlackConfiguration struct {
	WebhookUrl string `yaml:"webhookUrl"`
}

type RecoveryConfiguration struct {
	Enabled             bool                   `yaml:"enabled"`
	DayNumber           int                    `yaml:"dayNumber"`
	IntervalSecond      int                    `yaml:"intervalSecond"`
	PromQueryTimeSecond int64                  `yaml:"promQueryTimeSecond"`
	DingTalk            *DingTalkConfiguration `yaml:"dingTalk"`
	Slack               *SlackConfiguration    `yaml:"slack"`
}

type ModularConfiguration struct {
	Enabled             bool                   `yaml:"enabled"`
	CordonDailyLimit    map[string]int         `yaml:"cordonDailyLimit"`
	AbnormalityQL       map[string]string      `yaml:"abnormalityQL"`
	RecoveryQL          map[string]string      `yaml:"recoveryQL"`
	IntervalSecond      int                    `yaml:"intervalSecond"`
	PromQueryTimeSecond int64                  `yaml:"promQueryTimeSecond"`
	KubeMultiple        map[string]string      `yaml:"kubeMultiple"`
	DingTalk            *DingTalkConfiguration `yaml:"dingTalk"`
	Slack               *SlackConfiguration    `yaml:"slack"`
}

type DowntimeConfiguration struct {
	Enabled              bool                   `yaml:"enabled"`
	IntervalSecond       int                    `yaml:"intervalSecond"`
	PromQueryTimeSecond  int64                  `yaml:"promQueryTimeSecond"`
	KubeMultiple         map[string]string      `yaml:"kubeMultiple"`
	AbnormalityQL        []string               `yaml:"abnormalityQL"`
	AbnormalInfoSystemQL string                 `yaml:"abnormalInfoSystemQL"`
	DingTalk             *DingTalkConfiguration `yaml:"dingTalk"`
}

type AbnormalConfiguration struct {
	Enabled        bool                   `yaml:"enabled"`
	IntervalSecond int                    `yaml:"intervalSecond"`
	DoubleSecond   int                    `yaml:"doubleSecond"`
	LabelSelector  string                 `yaml:"labelSelector"`
	FieldSelector  string                 `yaml:"fieldSelector"`
	KubeMultiple   map[string]string      `yaml:"kubeMultiple"`
	DingTalk       *DingTalkConfiguration `yaml:"dingTalk"`
}
