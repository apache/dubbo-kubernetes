package manager_test

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/test"
)

func TestSecretManager(t *testing.T) {
	test.RunSpecs(t, "Secret Manager Suite")
}
