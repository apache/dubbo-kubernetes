package lookup_test

import (
	"testing"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/test"
)

func TestDNSCaching(t *testing.T) {
	test.RunSpecs(t, "DNS with cache Suite")
}
