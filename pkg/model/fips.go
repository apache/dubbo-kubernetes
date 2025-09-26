package model

import (
	gotls "crypto/tls"
	common_features "github.com/apache/dubbo-kubernetes/pkg/features"
	"k8s.io/klog/v2"
)

var fipsGoCiphers = []uint16{
	gotls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	gotls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	gotls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	gotls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

func EnforceGoCompliance(ctx *gotls.Config) {
	switch common_features.CompliancePolicy {
	case "":
		return
	case common_features.FIPS_140_2:
		ctx.MinVersion = gotls.VersionTLS12
		ctx.MaxVersion = gotls.VersionTLS12
		ctx.CipherSuites = fipsGoCiphers
		ctx.CurvePreferences = []gotls.CurveID{gotls.CurveP256}
		return
	case common_features.PQC:
		ctx.MinVersion = gotls.VersionTLS13
		ctx.CipherSuites = []uint16{gotls.TLS_AES_128_GCM_SHA256, gotls.TLS_AES_256_GCM_SHA384}
		ctx.CurvePreferences = []gotls.CurveID{gotls.X25519MLKEM768}
	default:
		klog.Warningf("unknown compliance policy: %q", common_features.CompliancePolicy)
		return
	}
}
