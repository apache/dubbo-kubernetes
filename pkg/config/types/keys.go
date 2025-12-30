//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"github.com/pkg/errors"
)

type PublicKey struct {
	// ID of key used to issue token.
	KID string `json:"kid"`
	// File with a public key encoded in PEM format.
	KeyFile string `json:"keyFile,omitempty"`
	// Public key encoded in PEM format.
	Key string `json:"key,omitempty"`
}

type MeshedPublicKey struct {
	PublicKey
	Mesh string `json:"mesh"`
}

func (p PublicKey) Validate() error {
	if p.KID == "" {
		return errors.New(".KID is required")
	}
	if p.KeyFile == "" && p.Key == "" {
		return errors.New("either .KeyFile or .Key has to be defined")
	}
	if p.KeyFile != "" && p.Key != "" {
		return errors.New("both .KeyFile or .Key cannot be defined")
	}
	return nil
}

func (m MeshedPublicKey) Validate() error {
	if err := m.PublicKey.Validate(); err != nil {
		return err
	}
	return nil
}
