package features

import "github.com/apache/dubbo-kubernetes/pkg/env"

var (
	EnableUnsafeAssertions = env.Register(
		"UNSAFE_NAVIGATOR_ENABLE_RUNTIME_ASSERTIONS",
		false,
		"If enabled, addition runtime asserts will be performed. "+
			"These checks are both expensive and panic on failure. As a result, this should be used only for testing.",
	).Get()
)
