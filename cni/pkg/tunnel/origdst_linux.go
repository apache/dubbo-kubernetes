//go:build linux

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

package tunnel

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"unsafe"
)

const soOriginalDst = 80 // Linux SO_ORIGINAL_DST

// OriginalDestination recovers the pre-NAT destination IP:port from a redirected Linux TCP connection.
func OriginalDestination(conn net.Conn) (string, error) {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return "", fmt.Errorf("connection is not *net.TCPConn")
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return "", err
	}
	var addr string
	var sockErr error
	err = rawConn.Control(func(fd uintptr) {
		var raw syscall.RawSockaddrInet4
		size := uint32(unsafe.Sizeof(raw))
		_, _, errno := syscall.Syscall6(
			syscall.SYS_GETSOCKOPT,
			fd,
			uintptr(syscall.SOL_IP),
			uintptr(soOriginalDst),
			uintptr(unsafe.Pointer(&raw)),
			uintptr(unsafe.Pointer(&size)),
			0,
		)
		if errno != 0 {
			sockErr = errno
			return
		}
		port := binary.BigEndian.Uint16((*[2]byte)(unsafe.Pointer(&raw.Port))[:])
		ip := net.IPv4(raw.Addr[0], raw.Addr[1], raw.Addr[2], raw.Addr[3])
		addr = net.JoinHostPort(ip.String(), fmt.Sprintf("%d", port))
	})
	if err != nil {
		return "", err
	}
	if sockErr != nil {
		return "", fmt.Errorf("getsockopt SO_ORIGINAL_DST: %w", sockErr)
	}
	return addr, nil
}
