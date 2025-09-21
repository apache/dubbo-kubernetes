package credentials

// CertInfo wraps a certificate, key, and oscp staple information.
type CertInfo struct {
	// The certificate chain
	Cert []byte
	// The private key
	Key []byte
	// The oscp staple
	Staple []byte
	// Certificate Revocation List information
	CRL []byte
}
