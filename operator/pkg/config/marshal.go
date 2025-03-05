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

package config

import (
	"errors"
	"github.com/golang/protobuf/jsonpb"
	legacyproto "github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/proto"
)

func MarshalIndent(msg proto.Message, indent string) ([]byte, error) {
	res, err := ToJSONWithIndent(msg, indent)
	if err != nil {
		return nil, err
	}
	return []byte(res), err
}

func ToJSONWithIndent(msg proto.Message, indent string) (string, error) {
	return ToJSONWithOptions(msg, indent, false)
}

func ToJSONWithOptions(msg proto.Message, indent string, enumsAsInts bool) (string, error) {
	if msg == nil {
		return "", errors.New("unexpected nil message")
	}
	m := jsonpb.Marshaler{Indent: indent, EnumsAsInts: enumsAsInts}
	return m.MarshalToString(legacyproto.MessageV1(msg))
}
