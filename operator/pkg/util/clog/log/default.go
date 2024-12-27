package log

func registerDefaultScopes() (defaults *Scope) {
	return registerScope(DefaultScopeName, 1)
}

var defaultScope = registerDefaultScopes()
