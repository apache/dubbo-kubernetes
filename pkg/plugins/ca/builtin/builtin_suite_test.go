package builtin_test

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/test"
)

func TestCaBuiltin(t *testing.T) {
	test.RunSpecs(t, "CA Builtin Suite")
}
