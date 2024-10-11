package v3_test

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/test"
)

func TestTLS(t *testing.T) {
	test.RunSpecs(t, "Envoy TLS v3 Suite")
}
