//go:build !noklog
// +build !noklog

package log

import (
	"k8s.io/klog/v2"
)

func init() {
	// Automatically intercept klog output when this package is imported
	setupKlogInterceptor()
}

// setupKlogInterceptor sets up klog interception
func setupKlogInterceptor() {
	interceptor := GetInterceptor()
	klog.SetOutput(interceptor.Writer())
}
