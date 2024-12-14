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

package common

//import (
//	"fmt"
//	"github.com/apache/dubbo-kubernetes/pkg/core"
//	"github.com/ory/viper"
//	"github.com/spf13/cobra"
//	"os"
//	"path/filepath"
//	"strings"
//)
//
//var ControlPlaneLog = core.Log.WithName("dubboctl")
//
//// bindFunc which conforms to the cobra PreRunE method signature
//type bindFunc func(*cobra.Command, []string) error
//
//// BindEnv returns a bindFunc that binds env vars to the named flags.
//func BindEnv(flags ...string) bindFunc {
//	return func(cmd *cobra.Command, args []string) (err error) {
//		for _, flag := range flags {
//			if err = viper.BindPFlag(flag, cmd.Flags().Lookup(flag)); err != nil {
//				return
//			}
//		}
//		viper.AutomaticEnv()        // read in environment variables for DUBBO_<flag>
//		viper.SetEnvPrefix("dubbo") // ensure that all have the prefix
//		return
//	}
//}
//
//func WriteFile(filename string, data []byte, perm os.FileMode) error {
//	if err := os.MkdirAll(filepath.Dir(filename), perm); err != nil {
//		return err
//	}
//	return os.WriteFile(filename, data, perm)
//}
//
//// AddConfirmFlag ensures common text/wording when the --path flag is used
//func AddConfirmFlag(cmd *cobra.Command, dflt bool) {
//	cmd.Flags().BoolP("confirm", "c", dflt, "Prompt to confirm options interactively ($DUBBO_CONFIRM)")
//}
//
//// AddPathFlag ensures common text/wording when the --path flag is used
//func AddPathFlag(cmd *cobra.Command) {
//	cmd.Flags().StringP("path", "p", "", "Path to the application.  Default is current directory ($DUBBO_PATH)")
//}
//
//// SurveySelectDefault returns 'value' if defined and exists in 'options'.
//// Otherwise, options[0] is returned if it exists.  Empty string otherwise.
////
//// Usage Example:
////
////	languages := []string{ "go", "node", "rust" },
////	survey.Select{
////	  Options: options,
////	  Default: surveySelectDefaut(cfg.Language, languages),
////	}
////
//// Summary:
////
//// This protects against an incorrectly initialized survey.Select when the user
//// has provided a nonexistant option (validation is handled elsewhere) or
//// when a value is required but there exists no defaults (no default value on
//// the associated flag).
////
//// Explanation:
////
//// The above example chooses the default for the Survey (--confirm) question
//// in a way that works with user-provided flag and environment variable values.
////
////	`cfg.Language` is the current value set in the config struct, which is
////	   populated from (in ascending order of precedence):
////	   static flag default, associated environment variable, or command flag.
////	`languages` are the options which are being used by the survey select.
////
//// This cascade allows for the Survey questions to be properly pre-initialzed
//// with their associated environment variables or flags.  For example,
//// A user whose default language is set to 'node' using the global environment
//// variable FUNC_LANGUAGE will have that option pre-selected when running
//// `dubbo create -c`.
////
//// The 'survey' package expects the value of the Default member to exist
//// in the 'Options' member.  This is not possible when user-provided data is
//// allowed for the default, hence this logic is necessary.
////
//// For example, when the user is using prompts (--confirm) to select from a set
//// of options, but the associated flag either has an unrecognized value, or no
//// value at all, without this logic the resulting select prompt would be
//// initialized with this as the default value, and the act of what appears to
//// be choose the first option displayed does not overwrite the invalid default.
//// It could perhaps be argued this is a shortcoming in the survey package, but
//// it is also clearly an error to provide invalid data for a default.
//func SurveySelectDefault(value string, options []string) string {
//	for _, v := range options {
//		if value == v {
//			return v // The provided value is acceptable
//		}
//	}
//	if len(options) > 0 {
//		return options[0] // Sync with the option which will be shown by the UX
//	}
//	// Either the value is not an option or there are no options.  Either of
//	// which should fail proper validation
//	return ""
//}
//
//// DeriveNameAndAbsolutePathFromPath returns application name and absolute path
//// to the application project root. The input parameter path could be one of:
//// 'relative/path/to/foo', '/absolute/path/to/foo', 'foo' or ”.
//func DeriveNameAndAbsolutePathFromPath(path string) (string, string) {
//	var absPath string
//
//	// If path is not specified, we would like to use current working dir
//	if path == "" {
//		path = cwd()
//	}
//
//	// Expand the passed function name to its absolute path
//	absPath, err := filepath.Abs(path)
//	if err != nil {
//		return "", ""
//	}
//
//	// Get the name of the function, which equals to name of the current directory
//	pathParts := strings.Split(strings.TrimRight(path, string(os.PathSeparator)), string(os.PathSeparator))
//	return pathParts[len(pathParts)-1], absPath
//}
//
//// cwd returns the current working directory or exits 1 printing the error.
//func cwd() (cwd string) {
//	cwd, err := os.Getwd()
//	if err != nil {
//		panic(fmt.Sprintf("Unable to determine current working directory: %v", err))
//	}
//	return cwd
//}
