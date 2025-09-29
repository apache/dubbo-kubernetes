package authenticate

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-kubernetes/pkg/config/mesh"
	"github.com/apache/dubbo-kubernetes/pkg/security"
	"github.com/apache/dubbo-kubernetes/pkg/spiffe"
	oidc "github.com/coreos/go-oidc/v3/oidc"
	"istio.io/api/security/v1beta1"
	"strings"
)

const (
	IDTokenAuthenticatorType = "IDTokenAuthenticator"
)

type JwtPayload struct {
	Aud []string `json:"aud"`
	Exp int      `json:"exp"`
	Iss string   `json:"iss"`
	Sub string   `json:"sub"`
}

type JwtAuthenticator struct {
	// holder of a mesh configuration for dynamically updating trust domain
	meshHolder mesh.Holder
	audiences  []string
	verifier   *oidc.IDTokenVerifier
}

func NewJwtAuthenticator(jwtRule *v1beta1.JWTRule, meshWatcher mesh.Watcher) (*JwtAuthenticator, error) {
	issuer := jwtRule.GetIssuer()
	jwksURL := jwtRule.GetJwksUri()
	// The key of a JWT issuer may change, so the key may need to be updated.
	// Based on https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc#NewRemoteKeySet
	// the oidc library handles caching and cache invalidation. Thus, the verifier
	// is only created once in the constructor.
	var verifier *oidc.IDTokenVerifier
	if len(jwksURL) == 0 {
		// OIDC discovery is used if jwksURL is not set.
		provider, err := oidc.NewProvider(context.Background(), issuer)
		// OIDC discovery may fail, e.g. http request for the OIDC server may fail.
		if err != nil {
			return nil, fmt.Errorf("failed at creating an OIDC provider for %v: %v", issuer, err)
		}
		verifier = provider.Verifier(&oidc.Config{SkipClientIDCheck: true})
	} else {
		keySet := oidc.NewRemoteKeySet(context.Background(), jwksURL)
		verifier = oidc.NewVerifier(issuer, keySet, &oidc.Config{SkipClientIDCheck: true})
	}
	return &JwtAuthenticator{
		meshHolder: meshWatcher,
		verifier:   verifier,
		audiences:  jwtRule.Audiences,
	}, nil
}

func (j *JwtAuthenticator) Authenticate(authRequest security.AuthContext) (*security.Caller, error) {
	if authRequest.GrpcContext != nil {
		bearerToken, err := security.ExtractBearerToken(authRequest.GrpcContext)
		if err != nil {
			return nil, fmt.Errorf("ID token extraction error: %v", err)
		}
		return j.authenticate(authRequest.GrpcContext, bearerToken)
	}
	if authRequest.Request != nil {
		bearerToken, err := security.ExtractRequestToken(authRequest.Request)
		if err != nil {
			return nil, fmt.Errorf("target JWT extraction error: %v", err)
		}
		return j.authenticate(authRequest.Request.Context(), bearerToken)
	}
	return nil, nil
}

func (j *JwtAuthenticator) authenticate(ctx context.Context, bearerToken string) (*security.Caller, error) {
	idToken, err := j.verifier.Verify(ctx, bearerToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify the JWT token (error %v)", err)
	}

	sa := JwtPayload{}
	// "aud" for trust domain, "sub" has "system:serviceaccount:$namespace:$serviceaccount".
	// in future trust domain may use another field as a standard is defined.
	if err := idToken.Claims(&sa); err != nil {
		return nil, fmt.Errorf("failed to extract claims from ID token: %v", err)
	}
	if !strings.HasPrefix(sa.Sub, "system:serviceaccount") {
		return nil, fmt.Errorf("invalid sub %v", sa.Sub)
	}
	parts := strings.Split(sa.Sub, ":")
	ns := parts[2]
	ksa := parts[3]
	if !checkAudience(sa.Aud, j.audiences) {
		return nil, fmt.Errorf("invalid audiences %v", sa.Aud)
	}
	return &security.Caller{
		AuthSource: security.AuthSourceIDToken,
		Identities: []string{spiffe.MustGenSpiffeURI(j.meshHolder.Mesh(), ns, ksa)},
	}, nil
}

func (j JwtAuthenticator) AuthenticatorType() string {
	return IDTokenAuthenticatorType
}

func checkAudience(audToCheck []string, audExpected []string) bool {
	for _, a := range audToCheck {
		for _, b := range audExpected {
			if a == b {
				return true
			}
		}
	}
	return false
}
