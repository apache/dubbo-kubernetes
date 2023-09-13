// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// LoadTemplate gets template content by the specified file.
func LoadTemplate(path, file, builtin string) (string, error) {
	file = filepath.Join(path, file)
	if !FileExists(file) {
		return builtin, nil
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func FileExists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
}

// FileNameWithoutExt returns a file name without suffix.
func FileNameWithoutExt(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file))
}

// EffectivePath to use is that which was provided by --path or FUNC_PATH.
// Manually parses flags such that this can be used during (cobra/viper) flag
// definition (prior to parsing).
func EffectivePath() (path string) {
	var (
		env = os.Getenv("DUBBO_PATH")
		fs  = pflag.NewFlagSet("", pflag.ContinueOnError)
		p   = fs.StringP("path", "p", "", "")
	)
	fs.SetOutput(io.Discard)
	fs.ParseErrorsWhitelist.UnknownFlags = true // wokeignore:rule=whitelist
	// Preparing flags intentionally ignores errors because this is intended
	// to be an opportunistic parse of the path flags, with actual validation of
	// flags taking place later in the instantiation process by the cobra pkg.
	_ = fs.Parse(os.Args[1:])
	if env != "" {
		path = env
	}
	if *p != "" {
		path = *p
	}
	return path
}

// InteractiveTerminal returns whether or not the currently attached process
// terminal is interactive.  Used for determining whether or not to
// interactively prompt the user to confirm default choices, etc.
func InteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
