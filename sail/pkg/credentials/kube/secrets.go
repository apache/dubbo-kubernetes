package kube

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/sail/pkg/credentials"
	"sort"
	"strings"
)

const (
	GenericScrtCaCert = "cacert"
	GenericScrtCRL    = "crl"

	TLSSecretCaCert = "ca.crt"
	TLSSecretCrl    = "ca.crl"
)

func hasKeys(d map[string][]byte, keys ...string) bool {
	for _, k := range keys {
		_, f := d[k]
		if !f {
			return false
		}
	}
	return true
}

func hasValue(d map[string][]byte, keys ...string) bool {
	for _, k := range keys {
		v := d[k]
		if len(v) == 0 {
			return false
		}
	}
	return true
}

func truncatedKeysMessage(data map[string][]byte) string {
	keys := []string{}
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) < 3 {
		return strings.Join(keys, ", ")
	}
	return fmt.Sprintf("%s, and %d more...", strings.Join(keys[:3], ", "), len(keys)-3)
}

// ExtractRoot extracts the root certificate
func ExtractRoot(data map[string][]byte) (certInfo *credentials.CertInfo, err error) {
	ret := &credentials.CertInfo{}
	if hasValue(data, GenericScrtCaCert) {
		ret.Cert = data[GenericScrtCaCert]
		ret.CRL = data[GenericScrtCRL]
		return ret, nil
	}
	if hasValue(data, TLSSecretCaCert) {
		ret.Cert = data[TLSSecretCaCert]
		ret.CRL = data[TLSSecretCrl]
		return ret, nil
	}
	// No cert found. Try to generate a helpful error message
	if hasKeys(data, GenericScrtCaCert) {
		return nil, fmt.Errorf("found key %q, but it was empty", GenericScrtCaCert)
	}
	if hasKeys(data, TLSSecretCaCert) {
		return nil, fmt.Errorf("found key %q, but it was empty", TLSSecretCaCert)
	}
	found := truncatedKeysMessage(data)
	return nil, fmt.Errorf("found secret, but didn't have expected keys %s or %s; found: %s",
		GenericScrtCaCert, TLSSecretCaCert, found)
}
