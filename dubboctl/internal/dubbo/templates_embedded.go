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

package dubbo

import (
	"archive/zip"
	"bytes"
)

import (
	"github.com/apache/dubbo-kubernetes/dubboctl/generated"
	"github.com/apache/dubbo-kubernetes/dubboctl/internal/filesystem"
)

//go:generate go run ../../generated/templates/generate.go
func newEmbeddedTemplatesFS() filesystem.Filesystem {
	archive, err := zip.NewReader(bytes.NewReader(generated.TemplatesZip), int64(len(generated.TemplatesZip)))
	if err != nil {
		panic(err)
	}
	return filesystem.NewZipFS(archive)
}

var EmbeddedTemplatesFS = newEmbeddedTemplatesFS()
