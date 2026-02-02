package main

import (
	"fmt"
	"os"

	"github.com/apache/dubbo-kubernetes/pkg/config/schema/codegen"
)

// Utility for generating collections.gen.go. Called from gen.go
func main() {
	if err := codegen.Run(); err != nil {
		fmt.Println(err)
		os.Exit(-2)
	}
}
