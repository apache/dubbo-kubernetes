//
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

package multierror

import (
	"strings"

	"github.com/hashicorp/go-multierror"
)

func MultiErrorFormat() multierror.ErrorFormatFunc {
	return func(es []error) string {
		if len(es) == 1 {
			return es[0].Error()
		}

		var b strings.Builder

		for i, err := range es {
			if i > 0 {
				b.WriteString("\n\t")
			}
			b.WriteString("* ")
			b.WriteString(err.Error())
		}

		b.WriteByte('\n')
		return b.String()
	}
}

func New() *multierror.Error {
	return &multierror.Error{ErrorFormat: MultiErrorFormat()}
}
