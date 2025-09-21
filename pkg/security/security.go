/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package security

import (
	"context"
	"net/http"
	"time"
)

const (
	// RootCertReqResourceName is resource name of discovery request for root certificate.
	RootCertReqResourceName = "ROOTCA"
	// WorkloadKeyCertResourceName is the resource name of the discovery request for workload
	// identity.
	WorkloadKeyCertResourceName = "default"
)

type AuthContext struct {
	// grpc context
	GrpcContext context.Context
	// http request
	Request *http.Request
}

type Authenticator interface {
	Authenticate(ctx AuthContext) (*Caller, error)
	AuthenticatorType() string
}

// SecretItem is the cached item in in-memory secret store.
type SecretItem struct {
	CertificateChain []byte
	PrivateKey       []byte

	RootCert []byte

	// ResourceName passed from envoy SDS discovery request.
	// "ROOTCA" for root cert request, "default" for key/cert request.
	ResourceName string

	CreatedTime time.Time

	ExpireTime time.Time
}

// SecretManager defines secrets management interface which is used by SDS.
type SecretManager interface {
	// GenerateSecret generates new secret for the given resource.
	//
	// The current implementation also watched the generated secret and trigger a callback when it is
	// near expiry. It will constructs the SAN based on the token's 'sub' claim, expected to be in
	// the K8S format. No other JWTs are currently supported due to client logic. If JWT is
	// missing/invalid, the resourceName is used.
	GenerateSecret(resourceName string) (*SecretItem, error)
}

type AuthSource int

type KubernetesInfo struct {
	PodName           string
	PodNamespace      string
	PodUID            string
	PodServiceAccount string
}

type Caller struct {
	AuthSource AuthSource
	Identities []string

	KubernetesInfo KubernetesInfo
}
