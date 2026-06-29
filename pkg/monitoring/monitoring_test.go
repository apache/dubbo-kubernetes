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

package monitoring

import (
	"fmt"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
)

func TestNewSumWithoutLabelsExposesCounter(t *testing.T) {
	name := fmt.Sprintf("test_sum_without_labels_total_%d", time.Now().UnixNano())
	metric := NewSum(name, "Test scalar counter.")

	metric.Increment()
	metric.RecordInt(2)

	family := findMetricFamily(t, name)
	if family.GetType() != dto.MetricType_COUNTER {
		t.Fatalf("metric type = %s, want COUNTER", family.GetType())
	}
	if got := family.GetMetric()[0].GetCounter().GetValue(); got != 3 {
		t.Fatalf("counter value = %v, want 3", got)
	}
}

func TestNewSumWithPredeclaredLabelsExposesCounterVec(t *testing.T) {
	phaseTag := CreateLabel("phase")
	name := fmt.Sprintf("test_sum_with_labels_total_%d", time.Now().UnixNano())
	metric := NewSum(name, "Test labeled counter.", WithLabels("phase"))

	metric.With(phaseTag.Value("push")).Increment()
	metric.With(phaseTag.Value("send")).RecordInt(2)

	assertCounterSeries(t, name, map[string]float64{
		"push": 1,
		"send": 2,
	})
}

func TestNewSumWithDynamicLabelsExposesCounterVec(t *testing.T) {
	typeTag := CreateLabel("type")
	name := fmt.Sprintf("test_sum_with_dynamic_labels_total_%d", time.Now().UnixNano())
	metric := NewSum(name, "Test dynamically labeled counter.")

	metric.With(typeTag.Value("cds")).Increment()
	metric.With(typeTag.Value("eds")).RecordInt(4)

	assertCounterSeries(t, name, map[string]float64{
		"cds": 1,
		"eds": 4,
	})
}

func TestDistributionWithLabelsRegistersOnce(t *testing.T) {
	phaseTag := CreateLabel("phase")
	name := fmt.Sprintf("test_distribution_with_labels_seconds_%d", time.Now().UnixNano())
	metric := NewDistribution(
		name,
		"Test labeled distribution registration.",
		[]float64{0.1, 1},
		WithLabels("phase"),
	)

	metric.With(phaseTag.Value("push")).Record(0.2)
	metric.With(phaseTag.Value("send")).Record(0.4)

	family := findMetricFamily(t, name)
	if got := len(family.GetMetric()); got != 2 {
		t.Fatalf("metric series = %d, want 2", got)
	}
}

func findMetricFamily(t *testing.T, name string) *dto.MetricFamily {
	t.Helper()
	families, err := GetRegistry().Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	t.Fatalf("%s not gathered", name)
	return nil
}

func assertCounterSeries(t *testing.T, name string, want map[string]float64) {
	t.Helper()
	family := findMetricFamily(t, name)
	if family.GetType() != dto.MetricType_COUNTER {
		t.Fatalf("metric type = %s, want COUNTER", family.GetType())
	}
	got := map[string]float64{}
	for _, metric := range family.GetMetric() {
		for _, label := range metric.GetLabel() {
			if label.GetName() == "phase" || label.GetName() == "type" {
				got[label.GetValue()] = metric.GetCounter().GetValue()
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("counter series = %#v, want %#v", got, want)
	}
	for label, value := range want {
		if got[label] != value {
			t.Fatalf("counter series %s = %v, want %v", label, got[label], value)
		}
	}
}
