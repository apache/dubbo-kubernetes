package ws_test

import (
	"testing"

	"github.com/apache/dubbo-kubernetes/pkg/test"
)

func TestWS(t *testing.T) {
	test.RunSpecs(t, "User Tokens WS Suite")
}
