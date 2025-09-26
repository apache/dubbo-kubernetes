package features

import (
	"github.com/apache/dubbo-kubernetes/pkg/env"
)

const (
	// FIPS_140_2 compliance policy.
	// nolint: revive, stylecheck
	FIPS_140_2 = "fips-140-2"

	PQC = "pqc"
)

// Define common security feature flags shared among the Istio components.
var (
	CompliancePolicy = env.Register("COMPLIANCE_POLICY", "",
		`If set, applies policy-specific restrictions over all existing TLS
settings, including in-mesh mTLS and external TLS. Valid values are:

* '' or unset places no additional restrictions.
* 'fips-140-2' which enforces a version of the TLS protocol and a subset
of cipher suites overriding any user preferences or defaults for all runtime
components, including Envoy, gRPC Go SDK, and gRPC C++ SDK.
* 'pqc' which enforces post-quantum-safe key exchange X25519MLKEM768, TLS v1.3
and cipher suites TLS_AES_128_GCM_SHA256 and TLS_AES_256_GCM_SHA384 overriding
any user preferences or defaults for all runtime components, including Envoy,
gRPC Go SDK, and gRPC C++ SDK. This policy is experimental.

WARNING: Setting compliance policy in the control plane is a necessary but
not a sufficient requirement to achieve compliance. There are additional
steps necessary to claim compliance, including using the validated
cryptograhic modules (please consult
https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/security/ssl#fips-140-2).`).Get()
)
