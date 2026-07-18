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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type PodState struct {
	ContainerID string `json:"containerID"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	IP          string `json:"ip"`
}

type FileStateStore struct {
	dir string
}

func NewFileStateStore(dir string) *FileStateStore {
	return &FileStateStore{dir: dir}
}

func (s *FileStateStore) Write(state PodState) error {
	if state.ContainerID == "" {
		return fmt.Errorf("container ID is required")
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(state.ContainerID), data, 0o600)
}

func (s *FileStateStore) Read(containerID string) (PodState, error) {
	data, err := os.ReadFile(s.path(containerID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return PodState{}, notFoundError{err}
		}
		return PodState{}, err
	}
	state := PodState{}
	if err := json.Unmarshal(data, &state); err != nil {
		return PodState{}, err
	}
	return state, nil
}

func (s *FileStateStore) Delete(containerID string) error {
	if err := os.Remove(s.path(containerID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (s *FileStateStore) path(containerID string) string {
	sum := sha256.Sum256([]byte(containerID))
	return filepath.Join(s.dir, hex.EncodeToString(sum[:])+".json")
}

type notFoundError struct {
	err error
}

func (e notFoundError) Error() string {
	return e.err.Error()
}

func (e notFoundError) Unwrap() error {
	return e.err
}

func IsNotFound(err error) bool {
	var nf notFoundError
	return errors.As(err, &nf)
}
