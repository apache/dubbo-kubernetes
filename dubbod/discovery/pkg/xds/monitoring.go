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

package xds

import (
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/model"
	v3 "github.com/apache/dubbo-kubernetes/dubbod/discovery/pkg/xds/v3"
	"github.com/apache/dubbo-kubernetes/pkg/monitoring"
)

var (
	typeTag    = monitoring.CreateLabel("type")
	versionTag = monitoring.CreateLabel("version")

	monServices = monitoring.NewGauge(
		"dubbod_services",
		"Total services known to dubbod.",
	)

	xdsClients = monitoring.NewGauge(
		"dubbod_xds",
		"Number of endpoints connected to this dubbod using XDS.",
		monitoring.WithLabels("version"),
	)
	xdsClientTrackerMutex = &sync.Mutex{}
	xdsClientTracker      = make(map[string]float64)

	pushes = monitoring.NewSum(
		"dubbod_xds_pushes",
		"Dubbod build and send errors for lds, rds, cds and eds.",
		monitoring.WithLabels("type"),
	)

	cdsSendErrPushes = pushes.With(typeTag.Value("cds_senderr"))
	edsSendErrPushes = pushes.With(typeTag.Value("eds_senderr"))
	ldsSendErrPushes = pushes.With(typeTag.Value("lds_senderr"))
	rdsSendErrPushes = pushes.With(typeTag.Value("rds_senderr"))

	debounceTime = monitoring.NewDistribution(
		"dubbod_debounce_time",
		"Delay in seconds between the first config enters debouncing and the merged push request is pushed into the push queue.",
		[]float64{.01, .1, 1, 3, 5, 10, 20, 30},
	)

	pushContextInitTime = monitoring.NewDistribution(
		"dubbod_pushcontext_init_seconds",
		"Total time in seconds Dubbod takes to init pushContext.",
		[]float64{.01, .1, 0.5, 1, 3, 5},
	)

	pushTime = monitoring.NewDistribution(
		"dubbod_xds_push_time",
		"Total time in seconds Dubbod takes to push lds, rds, cds and eds.",
		[]float64{.01, .1, 1, 3, 5, 10, 20, 30},
		monitoring.WithLabels("type"),
	)

	proxiesQueueTime = monitoring.NewDistribution(
		"dubbod_proxy_queue_time",
		"Time in seconds, a proxy is in the push queue before being dequeued.",
		[]float64{.1, .5, 1, 3, 5, 10, 20, 30},
	)

	pushTriggers = monitoring.NewSum(
		"dubbod_push_triggers",
		"Total number of times a push was triggered, labeled by reason for the push.",
		monitoring.WithLabels("type"),
	)

	proxiesConvergeDelay = monitoring.NewDistribution(
		"dubbod_proxy_convergence_time",
		"Delay in seconds between config change and a proxy receiving all required configuration.",
		[]float64{.1, .5, 1, 3, 5, 10, 20, 30},
	)

	inboundUpdates = monitoring.NewSum(
		"dubbod_inbound_updates",
		"Total number of updates received by dubbod.",
		monitoring.WithLabels("type"),
	)

	inboundConfigUpdates  = inboundUpdates.With(typeTag.Value("config"))
	inboundEDSUpdates     = inboundUpdates.With(typeTag.Value("eds"))
	inboundServiceUpdates = inboundUpdates.With(typeTag.Value("svc"))
	inboundServiceDeletes = inboundUpdates.With(typeTag.Value("svcdelete"))

	dubbodSDSCertificateErrors = monitoring.NewSum(
		"dubbod_sds_certificate_errors_total",
		"Total number of failures to fetch SDS key and certificate.",
	)

	configSizeBytes = monitoring.NewDistribution(
		"dubbod_xds_config_size_bytes",
		"Distribution of configuration sizes pushed to clients",
		[]float64{1, 10000, 1000000, 4000000, 10000000, 40000000},
		monitoring.WithUnit(monitoring.Bytes),
		monitoring.WithLabels("type"),
	)
)

func recordXDSClients(version string, delta float64) {
	xdsClientTrackerMutex.Lock()
	defer xdsClientTrackerMutex.Unlock()
	xdsClientTracker[version] += delta
	xdsClients.With(versionTag.Value(version)).Record(xdsClientTracker[version])
}

func recordPushTriggers(reasons model.ReasonStats) {
	for r, cnt := range reasons {
		pushTriggers.With(typeTag.Value(string(r))).RecordInt(int64(cnt))
	}
}

func isUnexpectedError(err error) bool {
	s, ok := status.FromError(err)
	isError := s.Code() != codes.Unavailable && s.Code() != codes.Canceled
	return !ok || isError
}

func recordSendError(xdsType string, err error) bool {
	if isUnexpectedError(err) {
		switch xdsType {
		case v3.ListenerType:
			ldsSendErrPushes.Increment()
		case v3.ClusterType:
			cdsSendErrPushes.Increment()
		case v3.EndpointType:
			edsSendErrPushes.Increment()
		case v3.RouteType:
			rdsSendErrPushes.Increment()
		}
		return true
	}
	return false
}

func recordPushTime(xdsType string, duration time.Duration) {
	metricType := v3.GetMetricType(xdsType)
	pushTime.With(typeTag.Value(metricType)).Record(duration.Seconds())
	pushes.With(typeTag.Value(metricType)).Increment()
}

func recordInboundConfigUpdate() {
	inboundConfigUpdates.Increment()
}

func recordInboundEDSUpdate() {
	inboundEDSUpdates.Increment()
}

func recordInboundServiceUpdate() {
	inboundServiceUpdates.Increment()
}

func recordInboundServiceDelete() {
	inboundServiceDeletes.Increment()
}
