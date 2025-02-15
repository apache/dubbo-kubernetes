// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generate

import (
	"fmt"
	"os"
)

import (
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

import (
	dubbo_cmd "github.com/apache/dubbo-kubernetes/pkg/core/cmd"
	"github.com/apache/dubbo-kubernetes/pkg/tls"
)

var NewSelfSignedCert = tls.NewSelfSignedCert

type generateCertificateContext struct {
	args struct {
		key       string
		cert      string
		certType  string
		keyType   string
		hostnames []string
	}
}

func NewGenerateCertificateCmd(baseCmd *cobra.Command) {
	ctx := &generateCertificateContext{}
	cmd := &cobra.Command{
		Use:   "tls-certificate",
		Short: "Generate a TLS certificate",
		Long:  `Generate self signed key and certificate pair that can be used for example in Dataplane Token Server setup.`,
		Example: `
  # Generate a TLS certificate for use by an HTTPS server, i.e. by the Dataplane Token server, Webhook Server
  dubboctl generate tls-certificate --type=server --hostname=localhost

  # Generate a TLS certificate for use by a client of an HTTPS server, i.e. by the 'dubboctl generate dataplane-token' command
  dubboctl generate tls-certificate --type=client --hostname=dataplane-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			certType := tls.CertType(ctx.args.certType)
			switch certType {
			case tls.ClientCertType, tls.ServerCertType:
				if len(ctx.args.hostnames) == 0 {
					return errors.New("at least one hostname must be given")
				}
			default:
				return errors.Errorf("invalid certificate type %q", certType)
			}

			keyType := tls.DefaultKeyType
			switch ctx.args.keyType {
			case "":
			case "rsa":
				keyType = tls.RSAKeyType
			case "ecdsa":
				keyType = tls.ECDSAKeyType
			default:
				return errors.Errorf("invalid key type %q", ctx.args.keyType)
			}

			keyPair, err := NewSelfSignedCert(certType, keyType, ctx.args.hostnames...)
			if err != nil {
				return errors.Wrap(err, "could not generate certificate")
			}

			if ctx.args.key == "-" {
				_, err = cmd.OutOrStdout().Write(keyPair.KeyPEM)
			} else {
				err = os.WriteFile(ctx.args.key, keyPair.KeyPEM, 0o400)
			}
			if err != nil {
				return errors.Wrap(err, "could not write the key file")
			}

			if ctx.args.cert == "-" {
				_, err = cmd.OutOrStdout().Write(keyPair.CertPEM)
			} else {
				err = os.WriteFile(ctx.args.cert, keyPair.CertPEM, 0o600)
			}
			if err != nil {
				return errors.Wrap(err, "could not write the cert file")
			}

			if ctx.args.cert != "-" && ctx.args.key != "-" {
				fmt.Fprintf(cmd.OutOrStdout(), "Private key saved in %s\n", ctx.args.key)
				fmt.Fprintf(cmd.OutOrStdout(), "Certificate saved in %s\n", ctx.args.cert)
			}

			return nil
		},
	}
	cmd.Flags().StringVar(&ctx.args.key, "key-file", "key.pem", "path to a file with a generated private key ('-' for stdout)")
	cmd.Flags().StringVar(&ctx.args.cert, "cert-file", "cert.pem", "path to a file with a generated TLS certificate ('-' for stdout)")
	cmd.Flags().StringVar(&ctx.args.certType, "type", "", dubbo_cmd.UsageOptions("type of the certificate", "client", "server"))
	cmd.Flags().StringVar(&ctx.args.keyType, "key-type", "", dubbo_cmd.UsageOptions("type of the private key", "rsa", "ecdsa"))
	cmd.Flags().StringSliceVar(&ctx.args.hostnames, "hostname", []string{}, "DNS hostname(s) to issue the certificate for")
	_ = cmd.MarkFlagRequired("type")
	_ = cmd.MarkFlagRequired("hostname")

	baseCmd.AddCommand(cmd)
}
