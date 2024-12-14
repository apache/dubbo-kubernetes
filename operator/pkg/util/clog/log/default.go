package log

func registerDefaultScopes() (defaults *Scope) {
	return registerScope(DefaultScopeName, "Unscoped logging messages.", 1)
}

var defaultScope = registerDefaultScopes()
