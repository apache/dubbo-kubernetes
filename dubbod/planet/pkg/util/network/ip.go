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

package network

import (
	"net"
	"net/netip"
	"time"

	"github.com/apache/dubbo-kubernetes/pkg/sleep"
	"golang.org/x/net/context"
)

const (
	waitInterval = 100 * time.Millisecond
	waitTimeout  = 2 * time.Minute
)

func AllIPv4(ipAddrs []string) bool {
	for i := 0; i < len(ipAddrs); i++ {
		addr, err := netip.ParseAddr(ipAddrs[i])
		if err != nil {
			// Should not happen, invalid IP in proxy's IPAddresses slice should have been caught earlier,
			// skip it to prevent a panic.
			continue
		}
		if !addr.Is4() && addr.Is6() {
			return false
		}
	}
	return true
}

func AllIPv6(ipAddrs []string) bool {
	for i := 0; i < len(ipAddrs); i++ {
		addr, err := netip.ParseAddr(ipAddrs[i])
		if err != nil {
			// Should not happen, invalid IP in proxy's IPAddresses slice should have been caught earlier,
			// skip it to prevent a panic.
			continue
		}
		if addr.Is4() {
			return false
		}
	}
	return true
}

func GlobalUnicastIP(ipAddrs []string) string {
	for i := 0; i < len(ipAddrs); i++ {
		addr, err := netip.ParseAddr(ipAddrs[i])
		if err != nil {
			// Should not happen, invalid IP in proxy's IPAddresses slice should have been caught earlier,
			// skip it to prevent a panic.
			continue
		}
		if addr.IsGlobalUnicast() {
			return addr.String()
		}
	}
	return ""
}

func GetPrivateIPs(ctx context.Context) ([]string, bool) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, waitTimeout)
		defer cancel()
	}

	for {
		select {
		case <-ctx.Done():
			return GetPrivateIPsIfAvailable()
		default:
			addr, ok := GetPrivateIPsIfAvailable()
			if ok {
				return addr, true
			}
			sleep.UntilContext(ctx, waitInterval)
		}
	}
}

func GetPrivateIPsIfAvailable() ([]string, bool) {
	ok := true
	ipAddresses := make([]string, 0)

	ifaces, _ := net.Interfaces()

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, _ := iface.Addrs()

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			default:
				continue

			}
			ipAddr, okay := netip.AddrFromSlice(ip)
			if !okay {
				continue
			}
			// unwrap the IPv4-mapped IPv6 address
			unwrapAddr := ipAddr.Unmap()
			if !unwrapAddr.IsValid() || unwrapAddr.IsLoopback() || unwrapAddr.IsLinkLocalUnicast() || unwrapAddr.IsLinkLocalMulticast() {
				continue
			}
			if unwrapAddr.IsUnspecified() {
				ok = false
				continue
			}
			ipAddresses = append(ipAddresses, unwrapAddr.String())
		}
	}
	return ipAddresses, ok
}
