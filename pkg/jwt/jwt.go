package jwt

const (
	PolicyFirstParty = "first-party-jwt"
	PolicyThirdParty = "third-party-jwt"
)

type JwksFetchMode int

const (
	// Istiod is used to indicate Istiod ALWAYS fetches the JWKs server
	Dubbod JwksFetchMode = iota
)

func (mode JwksFetchMode) String() string {
	switch mode {
	case Dubbod:
		return "Dubbod"
	default:
		return "Unset"
	}
}
