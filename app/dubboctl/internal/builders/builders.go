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

package builders

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/dubbo-kubernetes/app/dubboctl/internal/dubbo"
)

const (
	Pack = "pack(coming soon...)"
)

type Known []string

func (k Known) String() string {
	var b strings.Builder
	for i, v := range k {
		if i < len(k)-2 {
			b.WriteString(strconv.Quote(v) + ", ")
		} else if i < len(k)-1 {
			b.WriteString(strconv.Quote(v) + " and ")
		} else {
			b.WriteString(strconv.Quote(v))
		}
	}
	return b.String()
}

// ErrUnknownBuilder may be used by whomever is choosing a concrete
// implementation of a builder to invoke based on potentially invalid input.
type ErrUnknownBuilder struct {
	Name  string
	Known Known
}

func (e ErrUnknownBuilder) Error() string {
	if len(e.Known) == 0 {
		return fmt.Sprintf("\"%v\" is not a known builder", e.Name)
	}
	if len(e.Known) == 1 {
		return fmt.Sprintf("\"%v\" is not a known builder. The available builder is %v", e.Name, e.Known)
	}
	return fmt.Sprintf("\"%v\" is not a known builder. Available builders are %s", e.Name, e.Known)
}

type ErrBuilderNotSupported struct {
	Builder string
}

func (e ErrBuilderNotSupported) Error() string {
	return fmt.Sprintf("builder %q is not supported", e.Builder)
}

type ErrRuntimeRequired struct {
	Builder string
}

func (e ErrRuntimeRequired) Error() string {
	return fmt.Sprintf("runtime required to choose a default '%v' builder image", e.Builder)
}

type ErrNoDefaultImage struct {
	Builder string
	Runtime string
}

func (e ErrNoDefaultImage) Error() string {
	return fmt.Sprintf("the '%v' runtime defines no default '%v' builder image", e.Runtime, e.Builder)
}

// Image is a convenience function for choosing the correct builder image
// given a function, a builder, and defaults grouped by runtime.
//   - ErrRuntimeRequired if no runtime was provided on the given function
//   - ErrNoDefaultImage if the function has no builder image already defined
//     for the given runtime and there is no default in the provided map.
func Image(f dubbo.Dubbo, builder string, defaults map[string]string) (string, error) {
	v, ok := f.Build.BuilderImages[builder]
	if ok {
		return v, nil
	}
	if f.Runtime == "" {
		return "", ErrRuntimeRequired{Builder: builder}
	}
	v, ok = defaults[f.Runtime]
	if ok {
		return v, nil
	}
	return "", ErrNoDefaultImage{
		Builder: builder,
		Runtime: f.Runtime,
	}
}
