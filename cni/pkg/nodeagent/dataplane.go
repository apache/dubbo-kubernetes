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
	"fmt"
	"os"
	"strings"
)

// The inbound fence is a filter-table chain in the host network namespace, so
// it only sees traffic that actually traverses netfilter. Two common
// conditions silently bypass it: an eBPF dataplane that forwards in tc or XDP,
// and a bridged dataplane with bridge netfilter turned off. In both cases the
// rules install without error and match nothing, which looks identical to a
// working fence. These checks turn that into a visible failure.
const (
	bridgeNFCallIPTablesPath = "/proc/sys/net/bridge/bridge-nf-call-iptables"
	ciliumMarkerPath         = "/sys/class/net/cilium_host"
	ciliumConfigDir          = "/var/run/cilium"
)

// DataplaneWarning describes a condition under which the fence would not be
// enforced. It is an error type so callers can choose to fail startup.
type DataplaneWarning struct {
	Reason string
	Detail string
}

func (w DataplaneWarning) Error() string {
	if w.Detail == "" {
		return w.Reason
	}
	return w.Reason + ": " + w.Detail
}

// CheckDataplane reports conditions that would prevent the iptables fence from
// seeing pod traffic. A nil result means no known bypass was detected; it is
// not a guarantee that every dataplane routes through netfilter.
func CheckDataplane() []DataplaneWarning {
	warnings := []DataplaneWarning{}
	if detectEBPFDataplane() {
		warnings = append(warnings, DataplaneWarning{
			Reason: "an eBPF dataplane appears to be active",
			Detail: "pod traffic is forwarded in tc/XDP and never reaches the filter table, so the inbound fence will not be enforced",
		})
	}
	if enabled, known := bridgeNetfilterEnabled(); known && !enabled {
		warnings = append(warnings, DataplaneWarning{
			Reason: "bridge netfilter is disabled",
			Detail: bridgeNFCallIPTablesPath + " is 0, so same-node pod-to-pod traffic bypasses the inbound fence",
		})
	}
	if len(warnings) == 0 {
		return nil
	}
	return warnings
}

// VerifyDataplane returns an error when a bypass is detected, for callers that
// would rather refuse to start than run with a fence that does nothing.
func VerifyDataplane() error {
	warnings := CheckDataplane()
	if len(warnings) == 0 {
		return nil
	}
	reasons := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		reasons = append(reasons, warning.Error())
	}
	return fmt.Errorf("inbound fence would not be enforced on this node: %s", strings.Join(reasons, "; "))
}

// LogDataplaneWarnings reports detected bypasses without failing.
func LogDataplaneWarnings(out *os.File) {
	for _, warning := range CheckDataplane() {
		fmt.Fprintf(out, "dubbo-cni: warning: %s\n", warning.Error())
	}
}

func detectEBPFDataplane() bool {
	for _, path := range []string{ciliumMarkerPath, ciliumConfigDir} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// bridgeNetfilterEnabled reports the sysctl value and whether it could be
// read. An unreadable file usually means the bridge module is not loaded,
// which is normal on dataplanes that do not bridge, so it is not reported.
func bridgeNetfilterEnabled() (enabled bool, known bool) {
	data, err := os.ReadFile(bridgeNFCallIPTablesPath)
	if err != nil {
		return false, false
	}
	return strings.TrimSpace(string(data)) == "1", true
}
