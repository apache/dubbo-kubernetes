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

package protoconv

import (
	"github.com/apache/dubbo-kubernetes/sail/pkg/features"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func MessageToAnyWithError(msg proto.Message) (*anypb.Any, error) {
	b, err := marshal(msg)
	if err != nil {
		return nil, err
	}
	return &anypb.Any{
		// nolint: staticcheck
		TypeUrl: "type.googleapis.com/" + string(msg.ProtoReflect().Descriptor().FullName()),
		Value:   b,
	}, nil
}

func marshal(msg proto.Message) ([]byte, error) {
	if features.EnableVtprotobuf {
		if vt, ok := msg.(vtStrictMarshal); ok {
			// Attempt to use more efficient implementation
			// "Strict" is the equivalent to Deterministic=true below
			return vt.MarshalVTStrict()
		}
	}
	// If not available, fallback to normal implementation
	return proto.MarshalOptions{Deterministic: true}.Marshal(msg)
}

func MessageToAny(msg proto.Message) *anypb.Any {
	out, err := MessageToAnyWithError(msg)
	if err != nil {
		return nil
	}
	return out
}

type vtStrictMarshal interface {
	MarshalVTStrict() ([]byte, error)
}
