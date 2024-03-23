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

package golden

import (
	"fmt"
	"os"
	"path/filepath"
)

func UpdateGoldenFiles() bool {
	value, found := os.LookupEnv("UPDATE_GOLDEN_FILES")
	return found && value == "true"
}

func RerunMsg(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path + " Failed to retrieve cwd"
	}
	return fmt.Sprintf("Rerun the test with UPDATE_GOLDEN_FILES=true flag to update file: %s. Example: make test UPDATE_GOLDEN_FILES=true", absPath)
}
