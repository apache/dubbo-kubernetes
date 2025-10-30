package features

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	EnableCDSCaching = env.Register("SAIL_ENABLE_CDS_CACHE", true,
		"If true, SAIL will cache CDS responses. Note: this depends on SAIL_ENABLE_XDS_CACHE.").Get()

	EnableRDSCaching = env.Register("SAIL_ENABLE_RDS_CACHE", true,
		"If true, SAIL will cache RDS responses. Note: this depends on SAIL_ENABLE_XDS_CACHE.").Get()
)
