//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/monitoring"
)

func TestInfoJSONWritesIndentedJSONBlock(t *testing.T) {
	var out bytes.Buffer
	logger := RegisterScope("json-block-test", "json block test")
	logger.Scope().SetOutput(&out)

	logger.InfoJSON("Push Status", map[string]interface{}{
		"pushVersion": "v1",
		"clients": map[string]interface{}{
			"connectedEndpoints": 2,
		},
	})

	line := out.String()
	if !strings.Contains(line, "Push Status:\n{") {
		t.Fatalf("log line = %q, want title followed by JSON object", line)
	}
	if !strings.Contains(line, "\n  \"pushVersion\": \"v1\"") {
		t.Fatalf("log line = %q, want indented pushVersion field", line)
	}

	jsonStart := strings.Index(line, "{")
	if jsonStart < 0 {
		t.Fatalf("log line = %q, missing JSON object", line)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(line[jsonStart:]), &parsed); err != nil {
		t.Fatalf("indented block is not valid JSON: %v\n%s", err, line[jsonStart:])
	}
	if parsed["pushVersion"] != "v1" {
		t.Fatalf("pushVersion = %#v, want v1", parsed["pushVersion"])
	}
}

func TestLogMessagesMetricRecordsLevelAndScope(t *testing.T) {
	var out bytes.Buffer
	logger := RegisterScope("log-metric-test", "log metric test")
	logger.Scope().SetOutput(&out)

	logger.Warn("metric event")

	families, err := monitoring.GetRegistry().Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() != "dubbod_log_messages_total" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["level"] == "warn" && labels["scope"] == "log-metric-test" {
				if metric.GetGauge().GetValue() < 1 {
					t.Fatalf("dubbod_log_messages_total = %v, want >= 1", metric.GetGauge().GetValue())
				}
				return
			}
		}
	}
	t.Fatal("dubbod_log_messages_total with level=warn scope=log-metric-test not found")
}
