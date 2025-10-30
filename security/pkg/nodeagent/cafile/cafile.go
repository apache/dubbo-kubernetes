package cafile

import "github.com/apache/dubbo-kubernetes/pkg/security"

// CACertFilePath stores the OS CA certificate file path
var CACertFilePath = ""

func init() {
	CACertFilePath = security.GetOSRootFilePath()
}
