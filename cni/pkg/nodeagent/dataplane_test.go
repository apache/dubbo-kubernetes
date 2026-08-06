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

package nodeagent

import (
	"strings"
	"testing"
)

func TestDataplaneWarningError(t *testing.T) {
	warning := DataplaneWarning{Reason: "bridge netfilter is disabled", Detail: "same-node traffic bypasses the fence"}
	if got := warning.Error(); !strings.Contains(got, "bridge netfilter is disabled") || !strings.Contains(got, "bypasses") {
		t.Fatalf("Error() = %q, want reason and detail", got)
	}
	if got := (DataplaneWarning{Reason: "only a reason"}).Error(); got != "only a reason" {
		t.Fatalf("Error() = %q, want the bare reason", got)
	}
}

func TestVerifyDataplaneAgreesWithCheck(t *testing.T) {
	// The checks read host state, so assert the two entry points stay
	// consistent rather than pinning a result the CI host does not control.
	warnings := CheckDataplane()
	err := VerifyDataplane()
	if len(warnings) == 0 && err != nil {
		t.Fatalf("VerifyDataplane() = %v, want nil when no warning is reported", err)
	}
	if len(warnings) > 0 {
		if err == nil {
			t.Fatal("VerifyDataplane() = nil, want an error when warnings are reported")
		}
		for _, warning := range warnings {
			if !strings.Contains(err.Error(), warning.Reason) {
				t.Fatalf("VerifyDataplane() = %q, missing reason %q", err, warning.Reason)
			}
		}
	}
}
