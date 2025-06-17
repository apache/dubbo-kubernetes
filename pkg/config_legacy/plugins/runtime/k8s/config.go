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

package k8s

import (
	"time"
)

import (
	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

func DefaultKubernetesRuntimeConfig() *KubernetesRuntimeConfig {
	return &KubernetesRuntimeConfig{
		AdmissionServer: AdmissionServerConfig{
			Address: "10.23.132.51",
			Port:    5443,
		},
		MarshalingCacheExpirationTime: config_types.Duration{Duration: 5 * time.Minute},
		ControllersConcurrency: ControllersConcurrency{
			PodController: 10,
		},
		ClientConfig: ClientConfig{
			Qps:      100,
			BurstQps: 100,
		},
		LeaderElection: LeaderElection{
			LeaseDuration: config_types.Duration{Duration: 15 * time.Second},
			RenewDeadline: config_types.Duration{Duration: 10 * time.Second},
		},
	}
}

// KubernetesRuntimeConfig defines Kubernetes-specific configuration
type KubernetesRuntimeConfig struct {
	config.BaseConfig

	// Admission WebHook Server implemented by the Control Plane.
	AdmissionServer AdmissionServerConfig `json:"admissionServer"`
	// MarshalingCacheExpirationTime defines a duration for how long
	// marshaled objects will be stored in the cache. If equal to 0s then
	// cache is turned off
	MarshalingCacheExpirationTime config_types.Duration `json:"marshalingCacheExpirationTime" envconfig:"DUBBO_RUNTIME_KUBERNETES_MARSHALING_CACHE_EXPIRATION_TIME"`
	// Kubernetes' resources reconciliation concurrency configuration
	ControllersConcurrency ControllersConcurrency `json:"controllersConcurrency"`
	// Kubernetes client configuration
	ClientConfig ClientConfig `json:"clientConfig"`
	// Kubernetes leader election configuration
	LeaderElection LeaderElection `json:"leaderElection"`
}

type ControllersConcurrency struct {
	// PodController defines maximum concurrent reconciliations of Pod resources
	// Default value 10. If set to 0 kube controller-runtime default value of 1 will be used.
	PodController int `json:"podController" envconfig:"DUBBO_RUNTIME_KUBERNETES_CONTROLLERS_CONCURRENCY_POD_CONTROLLER"`
}

type ClientConfig struct {
	// Qps defines maximum requests kubernetes client is allowed to make per second.
	// Default value 100. If set to 0 kube-client default value of 5 will be used.
	Qps int `json:"qps" envconfig:"DUBBO_RUNTIME_KUBERNETES_CLIENT_CONFIG_QPS"`
	// BurstQps defines maximum burst requests kubernetes client is allowed to make per second
	// Default value 100. If set to 0 kube-client default value of 10 will be used.
	BurstQps       int    `json:"burstQps" envconfig:"DUBBO_RUNTIME_KUBERNETES_CLIENT_CONFIG_BURST_QPS"`
	KubeFileConfig string `json:"kube_file_config" envconfig:"DUBBO_RUNTIME_KUBE_FILE_CONFIG"`
}

type LeaderElection struct {
	// LeaseDuration is the duration that non-leader candidates will
	// wait to force acquire leadership. This is measured against time of
	// last observed ack. Default is 15 seconds.
	LeaseDuration config_types.Duration `json:"leaseDuration" envconfig:"DUBBO_RUNTIME_KUBERNETES_LEADER_ELECTION_LEASE_DURATION"`
	// RenewDeadline is the duration that the acting controlplane will retry
	// refreshing leadership before giving up. Default is 10 seconds.
	RenewDeadline config_types.Duration `json:"renewDeadline" envconfig:"DUBBO_RUNTIME_KUBERNETES_LEADER_ELECTION_RENEW_DEADLINE"`
}

// AdmissionServerConfig defines configuration of the Admission WebHook Server implemented by
// the Control Plane.
type AdmissionServerConfig struct {
	config.BaseConfig

	// Address the Admission WebHook Server should be listening on.
	Address string `json:"address" envconfig:"DUBBO_RUNTIME_KUBERNETES_ADMISSION_SERVER_ADDRESS"`
	// Port the Admission WebHook Server should be listening on.
	Port uint32 `json:"port" envconfig:"DUBBO_RUNTIME_KUBERNETES_ADMISSION_SERVER_PORT"`
	// Directory with a TLS cert and private key for the Admission WebHook Server.
	// TLS certificate file must be named `tls.crt`.
	// TLS key file must be named `tls.key`.
	CertDir string `json:"certDir" envconfig:"DUBBO_RUNTIME_KUBERNETES_ADMISSION_SERVER_CERT_DIR"`
}

var _ config.Config = &KubernetesRuntimeConfig{}

func (c *KubernetesRuntimeConfig) PostProcess() error {
	return multierr.Combine(
		c.AdmissionServer.PostProcess(),
	)
}

func (c *KubernetesRuntimeConfig) Validate() error {
	var errs error
	if err := c.AdmissionServer.Validate(); err != nil {
		errs = multierr.Append(errs, errors.Wrapf(err, ".AdmissionServer is not valid"))
	}
	if c.MarshalingCacheExpirationTime.Duration < 0 {
		errs = multierr.Append(errs, errors.Errorf(".MarshalingCacheExpirationTime must be positive or equal to 0"))
	}
	return errs
}

var _ config.Config = &AdmissionServerConfig{}

func (c *AdmissionServerConfig) Validate() error {
	var errs error
	if 65535 < c.Port {
		errs = multierr.Append(errs, errors.Errorf(".Port must be in the range [0, 65535]"))
	}
	if c.CertDir == "" {
		errs = multierr.Append(errs, errors.Errorf(".CertDir should not be empty"))
	}
	return errs
}

// DataplaneContainer defines the configuration of a Dubbo dataplane proxy container.
type DataplaneContainer struct {
	// Deprecated: Use DUBBO_BOOTSTRAP_SERVER_PARAMS_ADMIN_PORT instead.
	AdminPort uint32 `json:"adminPort,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_ADMIN_PORT"`
	// Drain time for listeners.
	DrainTime config_types.Duration `json:"drainTime,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_DRAIN_TIME"`
	// Readiness probe.
	ReadinessProbe SidecarReadinessProbe `json:"readinessProbe,omitempty"`
	// Liveness probe.
	LivenessProbe SidecarLivenessProbe `json:"livenessProbe,omitempty"`
	// EnvVars are additional environment variables that can be placed on Dubbo DP sidecar
	EnvVars map[string]string `json:"envVars" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_ENV_VARS"`
}

// SidecarReadinessProbe defines periodic probe of container service readiness.
type SidecarReadinessProbe struct {
	config.BaseConfig

	// Number of seconds after the container has started before readiness probes are initiated.
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_READINESS_PROBE_INITIAL_DELAY_SECONDS"`
	// Number of seconds after which the probe times out.
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_READINESS_PROBE_TIMEOUT_SECONDS"`
	// Number of seconds after which the probe times out.
	PeriodSeconds int32 `json:"periodSeconds,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_READINESS_PROBE_PERIOD_SECONDS"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	SuccessThreshold int32 `json:"successThreshold,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_READINESS_PROBE_SUCCESS_THRESHOLD"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	FailureThreshold int32 `json:"failureThreshold,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_READINESS_PROBE_FAILURE_THRESHOLD"`
}

// SidecarLivenessProbe defines periodic probe of container service liveness.
type SidecarLivenessProbe struct {
	config.BaseConfig

	// Number of seconds after the container has started before liveness probes are initiated.
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_LIVENESS_PROBE_INITIAL_DELAY_SECONDS"`
	// Number of seconds after which the probe times out.
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_LIVENESS_PROBE_TIMEOUT_SECONDS"`
	// How often (in seconds) to perform the probe.
	PeriodSeconds int32 `json:"periodSeconds,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_LIVENESS_PROBE_PERIOD_SECONDS"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	FailureThreshold int32 `json:"failureThreshold,omitempty" envconfig:"DUBBO_RUNTIME_KUBERNETES_INJECTOR_SIDECAR_CONTAINER_LIVENESS_PROBE_FAILURE_THRESHOLD"`
}
