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

// Code generated by protoc-gen-deepcopy. DO NOT EDIT.
package generated

import (
	proto "google.golang.org/protobuf/proto"
)

// DeepCopyInto supports using TagType within kubernetes types, where deepcopy-gen is used.
func (in *TagType) DeepCopyInto(out *TagType) {
	p := proto.Clone(in).(*TagType)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TagType. Required by controller-gen.
func (in *TagType) DeepCopy() *TagType {
	if in == nil {
		return nil
	}
	out := new(TagType)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new TagType. Required by controller-gen.
func (in *TagType) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using RepeatedFieldType within kubernetes types, where deepcopy-gen is used.
func (in *RepeatedFieldType) DeepCopyInto(out *RepeatedFieldType) {
	p := proto.Clone(in).(*RepeatedFieldType)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RepeatedFieldType. Required by controller-gen.
func (in *RepeatedFieldType) DeepCopy() *RepeatedFieldType {
	if in == nil {
		return nil
	}
	out := new(RepeatedFieldType)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new RepeatedFieldType. Required by controller-gen.
func (in *RepeatedFieldType) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}
