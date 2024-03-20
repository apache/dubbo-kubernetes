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

package files

import (
	"io/fs"
	"os"
	"path/filepath"
)

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func FileEmpty(path string) (bool, error) {
	file, err := os.Stat(path)
	if err != nil {
		return true, err
	}
	return file.Size() == 0, nil
}

// IsDirWriteable checks if dir is writable by writing and removing a file
// to dir. It returns nil if dir is writable.
func IsDirWriteable(dir string) error {
	f := filepath.Join(dir, ".touch")
	perm := 0o600
	if err := os.WriteFile(f, []byte(""), fs.FileMode(perm)); err != nil {
		return err
	}
	return os.Remove(f)
}
