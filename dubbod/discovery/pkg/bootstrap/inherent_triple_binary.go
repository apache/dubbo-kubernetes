// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type inherentTripleBinaryMetadata struct {
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type inherentTripleBinary struct {
	data     []byte
	metadata inherentTripleBinaryMetadata
}

var (
	inherentTripleBinariesOnce sync.Once
	inherentTripleBinaries     map[string]inherentTripleBinary
	inherentTripleBinariesErr  error
)

func loadInherentTripleBinaries() (map[string]inherentTripleBinary, error) {
	inherentTripleBinariesOnce.Do(func() {
		var files fs.FS = tripleEngineFiles
		if files == nil {
			inherentTripleBinariesErr = fmt.Errorf("dubbod was built without embedded Triple binaries; provide TRIPLE_ENGINE_BINARY_DIR when packaging dubbod")
			return
		}
		names, err := fs.Glob(files, "triple-binaries/build-*/*.bin")
		if err != nil || len(names) == 0 {
			inherentTripleBinariesErr = fmt.Errorf("embedded Triple binary bundle is empty: %v", err)
			return
		}
		binaries := make(map[string]inherentTripleBinary, len(names))
		for _, name := range names {
			platform := strings.TrimSuffix(path.Base(name), ".bin")
			var machine elf.Machine
			switch platform {
			case "linux-amd64":
				machine = elf.EM_X86_64
			case "linux-arm64":
				machine = elf.EM_AARCH64
			default:
				inherentTripleBinariesErr = fmt.Errorf("unsupported embedded Triple binary %q", name)
				return
			}
			data, err := fs.ReadFile(files, name)
			if err != nil {
				inherentTripleBinariesErr = fmt.Errorf("read embedded Triple binary: %w", err)
				return
			}
			executable, err := elf.NewFile(bytes.NewReader(data))
			if err != nil {
				inherentTripleBinariesErr = fmt.Errorf("invalid embedded Triple executable %q: %w", platform, err)
				return
			}
			valid := executable.Class == elf.ELFCLASS64 && executable.Data == elf.ELFDATA2LSB &&
				executable.Machine == machine && executable.Entry != 0 &&
				(executable.Type == elf.ET_EXEC || executable.Type == elf.ET_DYN)
			for _, program := range executable.Progs {
				if program.Type == elf.PT_INTERP {
					valid = false
				}
			}
			_ = executable.Close()
			if !valid {
				inherentTripleBinariesErr = fmt.Errorf("Triple binary %q must be a static Linux executable for its declared architecture", platform)
				return
			}
			digest := sha256.Sum256(data)
			binary := inherentTripleBinary{
				data: data,
				metadata: inherentTripleBinaryMetadata{
					SHA256: hex.EncodeToString(digest[:]),
					Size:   int64(len(data)),
				},
			}
			if previous, found := binaries[platform]; found && previous.metadata != binary.metadata {
				inherentTripleBinariesErr = fmt.Errorf("conflicting embedded Triple binaries for %q", platform)
				return
			}
			binaries[platform] = binary
		}
		inherentTripleBinaries = binaries
	})
	return inherentTripleBinaries, inherentTripleBinariesErr
}

// InherentTripleBinaryManifest describes the binaries carried by this controller.
// Executable bytes stay in dubbod; this manifest is available for explicit checks.
func InherentTripleBinaryManifest() ([]byte, error) {
	binaries, err := loadInherentTripleBinaries()
	if err != nil {
		return nil, err
	}
	manifest := make(map[string]inherentTripleBinaryMetadata, len(binaries))
	for platform, binary := range binaries {
		manifest[platform] = binary.metadata
	}
	return json.Marshal(manifest)
}

// ReleaseInherentTripleBinary releases the executable matching the workload
// architecture, checking it against a manifest when one is supplied.
func ReleaseInherentTripleBinary(directory, manifestFile string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Triple workload binary requires Linux")
	}
	binaries, err := loadInherentTripleBinaries()
	if err != nil {
		return err
	}
	platform := runtime.GOOS + "-" + runtime.GOARCH
	binary, found := binaries[platform]
	if !found {
		return fmt.Errorf("dubbod does not carry a Triple binary for %q", platform)
	}
	if manifestFile != "" {
		data, err := os.ReadFile(manifestFile)
		if err != nil {
			return fmt.Errorf("read controller Triple manifest: %w", err)
		}
		manifest := map[string]inherentTripleBinaryMetadata{}
		if err := json.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("decode controller Triple manifest: %w", err)
		}
		if expected, found := manifest[platform]; !found || expected != binary.metadata {
			return fmt.Errorf("Triple binary for %q does not match the controller manifest", platform)
		}
	}
	if directory == "" {
		return fmt.Errorf("Triple binary directory is empty")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create Triple binary directory: %w", err)
	}
	output, err := os.CreateTemp(directory, ".triple-engine-*")
	if err != nil {
		return fmt.Errorf("create Triple executable: %w", err)
	}
	defer func() {
		_ = output.Close()
		_ = os.Remove(output.Name())
	}()
	if _, err := output.Write(binary.data); err != nil {
		return fmt.Errorf("write Triple executable: %w", err)
	}
	if err := output.Chmod(0o555); err != nil {
		return fmt.Errorf("set Triple executable permissions: %w", err)
	}
	if err := output.Sync(); err != nil {
		return fmt.Errorf("sync Triple executable: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close Triple executable: %w", err)
	}
	if err := os.Rename(output.Name(), filepath.Join(directory, "triple-engine")); err != nil {
		return fmt.Errorf("publish Triple executable: %w", err)
	}
	return nil
}

// WaitForInherentTripleBootstrap waits for the projected Secret to replace the
// initial 0.0.0.0 node identity with the IP assigned to this workload.
func WaitForInherentTripleBootstrap(ctx context.Context, bootstrapFile, instanceIP string) error {
	expected := net.ParseIP(instanceIP)
	if expected == nil || expected.IsUnspecified() {
		return fmt.Errorf("Triple bootstrap wait requires the assigned workload IP")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var lastErr error
	for {
		ip, err := readInherentTripleNodeIP(bootstrapFile)
		if err == nil && ip.Equal(expected) {
			return nil
		}
		lastErr = err
		if lastErr == nil {
			lastErr = fmt.Errorf("bootstrap node IP does not yet match the workload IP")
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for Triple bootstrap %s: %w; %v", bootstrapFile, ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func readInherentTripleNodeIP(bootstrapFile string) (net.IP, error) {
	file, err := os.Open(bootstrapFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var bootstrap struct {
		Node struct {
			ID string `json:"id"`
		} `json:"node"`
	}
	if err := json.NewDecoder(io.LimitReader(file, 1<<20)).Decode(&bootstrap); err != nil {
		return nil, err
	}
	parts := strings.Split(bootstrap.Node.ID, "~")
	if len(parts) != 4 || parts[0] != "inherent" || net.ParseIP(parts[1]) == nil {
		return nil, fmt.Errorf("invalid inherent workload node identity in bootstrap")
	}
	return net.ParseIP(parts[1]), nil
}
