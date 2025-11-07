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

package k8sresourcelock

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LeaderElectionRecordAnnotationKey = "control-plane.alpha.kubernetes.io/leader"
)

type LeaderElectionRecord struct {
	HolderIdentity       string      `json:"holderIdentity"`
	HolderKey            string      `json:"holderKey"`
	LeaseDurationSeconds int         `json:"leaseDurationSeconds"`
	AcquireTime          metav1.Time `json:"acquireTime"`
	RenewTime            metav1.Time `json:"renewTime"`
	LeaderTransitions    int         `json:"leaderTransitions"`
}

type Interface interface {
	Get(ctx context.Context) (*LeaderElectionRecord, []byte, error)
	Create(ctx context.Context, ler LeaderElectionRecord) error
	Update(ctx context.Context, ler LeaderElectionRecord) error
	Identity() string
	Key() string
	Describe() string
	RecordEvent(string)
}

type ResourceLockConfig struct {
	Identity      string
	Key           string
	EventRecorder EventRecorder
}
