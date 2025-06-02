package main

import (
	"github.com/apache/dubbo-kubernetes/comet/galaxy/comet-discovery/app"
	"os"
)

func main() {
	rootCmd := app.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}
