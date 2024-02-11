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
	"github.com/apache/dubbo-kubernetes/pkg/core"
)

const defaultServiceAccountName = "system:serviceaccount:dubbo-system:dubbo-control-plane"

var logger = core.Log.WithName("kubernetes-config")

func DefaultKubernetesRuntimeConfig() *KubernetesRuntimeConfig {
	return &KubernetesRuntimeConfig{
		AdmissionServer: AdmissionServerConfig{
			Port: 5443,
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
	MarshalingCacheExpirationTime config_types.Duration `json:"marshalingCacheExpirationTime" envconfig:"dubbo_runtime_kubernetes_marshaling_cache_expiration_time"`
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
	PodController int `json:"podController" envconfig:"dubbo_runtime_kubernetes_controllers_concurrency_pod_controller"`
}

type ClientConfig struct {
	// Qps defines maximum requests kubernetes client is allowed to make per second.
	// Default value 100. If set to 0 kube-client default value of 5 will be used.
	Qps int `json:"qps" envconfig:"dubbo_runtime_kubernetes_client_config_qps"`
	// BurstQps defines maximum burst requests kubernetes client is allowed to make per second
	// Default value 100. If set to 0 kube-client default value of 10 will be used.
	BurstQps       int    `json:"burstQps" envconfig:"dubbo_runtime_kubernetes_client_config_burst_qps"`
	KubeFileConfig string `json:"kube_file_config" envconfig:"dubbo_runtime_kube_file_config"`
}

type LeaderElection struct {
	// LeaseDuration is the duration that non-leader candidates will
	// wait to force acquire leadership. This is measured against time of
	// last observed ack. Default is 15 seconds.
	LeaseDuration config_types.Duration `json:"leaseDuration" envconfig:"dubbo_runtime_kubernetes_leader_election_lease_duration"`
	// RenewDeadline is the duration that the acting controlplane will retry
	// refreshing leadership before giving up. Default is 10 seconds.
	RenewDeadline config_types.Duration `json:"renewDeadline" envconfig:"dubbo_runtime_kubernetes_leader_election_renew_deadline"`
}

// AdmissionServerConfig defines configuration of the Admission WebHook Server implemented by
// the Control Plane.
type AdmissionServerConfig struct {
	config.BaseConfig

	// Address the Admission WebHook Server should be listening on.
	Address string `json:"address" envconfig:"dubbo_runtime_kubernetes_admission_server_address"`
	// Port the Admission WebHook Server should be listening on.
	Port uint32 `json:"port" envconfig:"dubbo_runtime_kubernetes_admission_server_port"`
	// Directory with a TLS cert and private key for the Admission WebHook Server.
	// TLS certificate file must be named `tls.crt`.
	// TLS key file must be named `tls.key`.
	CertDir string `json:"certDir" envconfig:"dubbo_runtime_kubernetes_admission_server_cert_dir"`
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
	//if c.CertDir == "" {
	//	errs = multierr.Append(errs, errors.Errorf(".CertDir should not be empty"))
	//}
	return errs
}

// DataplaneContainer defines the configuration of a Kuma dataplane proxy container.
type DataplaneContainer struct {
	// Deprecated: Use DUBBO_BOOTSTRAP_SERVER_PARAMS_ADMIN_PORT instead.
	AdminPort uint32 `json:"adminPort,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_admin_port"`
	// Drain time for listeners.
	DrainTime config_types.Duration `json:"drainTime,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_drain_time"`
	// Readiness probe.
	ReadinessProbe SidecarReadinessProbe `json:"readinessProbe,omitempty"`
	// Liveness probe.
	LivenessProbe SidecarLivenessProbe `json:"livenessProbe,omitempty"`
	// EnvVars are additional environment variables that can be placed on Kuma DP sidecar
	EnvVars map[string]string `json:"envVars" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_env_vars"`
}

// SidecarReadinessProbe defines periodic probe of container service readiness.
type SidecarReadinessProbe struct {
	config.BaseConfig

	// Number of seconds after the container has started before readiness probes are initiated.
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_readiness_probe_initial_delay_seconds"`
	// Number of seconds after which the probe times out.
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_readiness_probe_timeout_seconds"`
	// Number of seconds after which the probe times out.
	PeriodSeconds int32 `json:"periodSeconds,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_readiness_probe_period_seconds"`
	// Minimum consecutive successes for the probe to be considered successful after having failed.
	SuccessThreshold int32 `json:"successThreshold,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_readiness_probe_success_threshold"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	FailureThreshold int32 `json:"failureThreshold,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_readiness_probe_failure_threshold"`
}

// SidecarLivenessProbe defines periodic probe of container service liveness.
type SidecarLivenessProbe struct {
	config.BaseConfig

	// Number of seconds after the container has started before liveness probes are initiated.
	InitialDelaySeconds int32 `json:"initialDelaySeconds,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_liveness_probe_initial_delay_seconds"`
	// Number of seconds after which the probe times out.
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_liveness_probe_timeout_seconds"`
	// How often (in seconds) to perform the probe.
	PeriodSeconds int32 `json:"periodSeconds,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_liveness_probe_period_seconds"`
	// Minimum consecutive failures for the probe to be considered failed after having succeeded.
	FailureThreshold int32 `json:"failureThreshold,omitempty" envconfig:"dubbo_runtime_kubernetes_injector_sidecar_container_liveness_probe_failure_threshold"`
}
