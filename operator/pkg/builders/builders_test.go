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

package builders_test

//import (
//	"errors"
//	"github.com/apache/dubbo-kubernetes/operator/dubbo"
//	"github.com/apache/dubbo-kubernetes/operator/pkg/builders"
//	"github.com/apache/dubbo-kubernetes/operator/pkg/builders/pack"
//	"testing"
//)
//
//// TestImage_Named ensures that a builder image is returned when
//// it exists on the function for a given builder, no defaults.
//func TestImage_Named(t *testing.T) {
//	f := &dubbo.Dubbo{
//		Build: dubbo.BuildSpec{
//			BuilderImages: map[string]string{
//				builders.Pack: "example.com/my/builder-image",
//			},
//		},
//	}
//
//	builderImage, err := builders.Image(f, builders.Pack, pack.DefaultBuilderImages)
//	if err != nil {
//		t.Fatal(err)
//	}
//	if builderImage != "example.com/my/builder-image" {
//		t.Fatalf("expected 'example.com/my/builder-image', got '%v'", builderImage)
//	}
//}
//
//// TestImage_ErrRuntimeRequired ensures that the correct error is thrown when
//// the function has no builder image yet defined for the named builder, and
//// also no runtime to choose from the defaults.
//func TestImage_ErrRuntimeRequired(t *testing.T) {
//	_, err := builders.Image(&dubbo.Dubbo{}, "", pack.DefaultBuilderImages)
//	if err == nil {
//		t.Fatalf("did not receive expected error")
//	}
//	if !errors.Is(err, builders.ErrRuntimeRequired{}) {
//		t.Fatalf("error is not an 'ErrRuntimeRequired': '%v'", err)
//	}
//}
//
//// TestImage_ErrNoDefaultImage ensures that when
//func TestImage_ErrNoDefaultImage(t *testing.T) {
//	d := &dubbo.Dubbo{
//		Runtime: "go",
//		Build: dubbo.BuildSpec{
//			BuilderImages: map[string]string{},
//		},
//	}
//	_, err := builders.Image(d, "", map[string]string{})
//	if err == nil {
//		t.Fatalf("did not receive expected error")
//	}
//	if !errors.Is(err, builders.ErrNoDefaultImage{Runtime: "go"}) {
//		t.Fatalf("did not get 'ErrNoDefaultImage', got '%v'", err)
//	}
//}
//
//// TestImage_Defaults ensures that, when a default exists in the provided
//// map, it is chosen when both runtime is defined on the function and no
//// builder image has yet to be defined on the function.
//func TestImage_Defaults(t *testing.T) {
//	d := &dubbo.Dubbo{
//		Runtime: "go",
//		Build: dubbo.BuildSpec{
//			BuilderImages: map[string]string{},
//		},
//	}
//	defaults := map[string]string{
//		"go": "example.com/go/default-builder-image",
//	}
//	builderImage, err := builders.Image(d, "", defaults)
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	if builderImage != "example.com/go/default-builder-image" {
//		t.Fatalf("the default was not chosen")
//	}
//}
//
//// Test_ErrUnknownBuilder ensures that the error properly formats.
//// This error is used externally by packages which share builders but may
//// define their own custom builder, thus actually throwing this error
//// is the responsibility of whomever is instantiating builders.
//func Test_ErrUnknownBuilder(t *testing.T) {
//	tests := []struct {
//		Known    []string
//		Expected string
//	}{
//		{
//			[]string{},
//			`"test" is not a known builder`,
//		},
//		{
//			[]string{"pack"},
//			`"test" is not a known builder. The available builder is "pack"`,
//		},
//		{
//			[]string{"pack", "s2i"},
//			`"test" is not a known builder. Available builders are "pack" and "s2i"`,
//		},
//		{
//			[]string{"pack", "s2i", "custom"},
//			`"test" is not a known builder. Available builders are "pack", "s2i" and "custom"`,
//		},
//	}
//	for _, test := range tests {
//		e := builders.ErrUnknownBuilder{Name: "test", Known: test.Known}
//		if e.Error() != test.Expected {
//			t.Fatalf("expected error \"%v\". got \"%v\"", test.Expected, e.Error())
//		}
//	}
//}
