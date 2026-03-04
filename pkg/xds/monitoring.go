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
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/apache/dubbo-kubernetes/pkg/log"
	"github.com/apache/dubbo-kubernetes/pkg/model"
	"github.com/apache/dubbo-kubernetes/pkg/monitoring"
)

var (
	Log = log.RegisterScope("ads", "ads debugging")
	xdsLog = Log

	errTag  = monitoring.CreateLabel("err")
	nodeTag = monitoring.CreateLabel("node")
	typeTag = monitoring.CreateLabel("type")

	TotalXDSInternalErrors = monitoring.NewSum(
		"dubbod_total_xds_internal_errors",
		"Total number of internal XDS errors in dubbod.",
	)

	ExpiredNonce = monitoring.NewSum(
		"dubbod_xds_expired_nonce",
		"Total number of XDS requests with an expired nonce.",
	)

	cdsReject = monitoring.NewGauge(
		"dubbod_xds_cds_reject",
		"Dubbod rejected CDS configs.",
	)

	edsReject = monitoring.NewGauge(
		"dubbod_xds_eds_reject",
		"Dubbod rejected EDS.",
	)

	ldsReject = monitoring.NewGauge(
		"dubbod_xds_lds_reject",
		"Dubbod rejected LDS.",
	)

	rdsReject = monitoring.NewGauge(
		"dubbod_xds_rds_reject",
		"Dubbod rejected RDS.",
	)

	totalXDSRejects = monitoring.NewSum(
		"dubbod_total_xds_rejects",
		"Total number of XDS responses from dubbod rejected by proxy.",
	)

	ResponseWriteTimeouts = monitoring.NewSum(
		"dubbod_xds_write_timeout",
		"Dubbod XDS response write timeouts.",
	)

	sendTime = monitoring.NewDistribution(
		"dubbod_xds_send_time",
		"Total time in seconds Dubbod takes to send generated configuration.",
		[]float64{.01, .1, 1, 3, 5, 10, 20, 30},
	)
)

func IncrementXDSRejects(xdsType string, node, errCode string) {
	metricType := model.GetShortType(xdsType)
	totalXDSRejects.With(typeTag.Value(metricType)).Increment()
	switch xdsType {
	case model.ListenerType:
		ldsReject.With(nodeTag.Value(node), errTag.Value(errCode)).Increment()
	case model.ClusterType:
		cdsReject.With(nodeTag.Value(node), errTag.Value(errCode)).Increment()
	case model.EndpointType:
		edsReject.With(nodeTag.Value(node), errTag.Value(errCode)).Increment()
	case model.RouteType:
		rdsReject.With(nodeTag.Value(node), errTag.Value(errCode)).Increment()
	}
}

func RecordSendTime(duration time.Duration) {
	sendTime.Record(duration.Seconds())
}

func RecordSendError(xdsType string, err error) {
	if isUnexpectedError(err) {
		TotalXDSInternalErrors.Increment()
	}
}

func isUnexpectedError(err error) bool {
	s, ok := status.FromError(err)
	isError := s.Code() != codes.Unavailable && s.Code() != codes.Canceled
	return !ok || isError
}
