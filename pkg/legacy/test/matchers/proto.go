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

package matchers

import (
	"github.com/google/go-cmp/cmp"

	"github.com/onsi/gomega/types"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/testing/protocmp"
)

func MatchProto(expected interface{}) types.GomegaMatcher {
	return &ProtoMatcher{
		Expected: expected,
	}
}

type ProtoMatcher struct {
	Expected interface{}
}

func (p *ProtoMatcher) Match(actual interface{}) (bool, error) {
	if actual == nil && p.Expected == nil {
		return true, nil
	}
	if actual == nil && p.Expected != nil {
		return false, errors.New("Actual object is nil, but Expected object is not.")
	}
	if actual != nil && p.Expected == nil {
		return false, errors.New("Actual object is not nil, but Expected object is.")
	}

	actualProto, ok := actual.(proto.Message)
	if !ok {
		return false, errors.New("You can only compare proto with this matcher. Make sure the object passed to MatchProto() implements proto.Message")
	}

	expectedProto, ok := p.Expected.(proto.Message)
	if !ok {
		return false, errors.New("You can only compare proto with this matcher. Make sure the object passed to Expect() implements proto.Message")
	}

	return proto.Equal(actualProto, expectedProto), nil
}

func (p *ProtoMatcher) FailureMessage(actual interface{}) string {
	differences := cmp.Diff(p.Expected, actual, protocmp.Transform())
	return "Expected matching protobuf message:\n" + differences
}

func (p *ProtoMatcher) NegatedFailureMessage(actual interface{}) string {
	return "Expected different protobuf but was the same"
}

var _ types.GomegaMatcher = &ProtoMatcher{}
