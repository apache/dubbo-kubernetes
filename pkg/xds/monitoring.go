package xds

import dubbolog "github.com/apache/dubbo-kubernetes/pkg/log"

var (
	Log = dubbolog.RegisterScope("ads", "ads debugging")
	log = Log
)
