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

package k8s

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestK8sNameCheck(t *testing.T) {
	assert.False(t, k8sNameCheck.MatchString("com.apache.ServiceName::.Suffix"))
	assert.False(t, k8sNameCheck.MatchString("AAAAA.condition-router"))
	assert.True(t, k8sNameCheck.MatchString("asdf.condition-router"))
}

func TestEncodeDecodeK8sName(t *testing.T) {
	var name string
	{
		name = "applicationName.Suffix"
		Name, err := EncodeK8sResName(name)
		assert.Nil(t, err)
		assert.True(t, k8sNameCheck.MatchString(Name))
		Name, err = DecodeK8sResName(Name)
		assert.Nil(t, err)
		assert.Equal(t, name, Name)
	}
	{
		name = "com.apache.ServiceName::.Suffix"
		Name, err := EncodeK8sResName(name)
		assert.Nil(t, err)
		assert.True(t, k8sNameCheck.MatchString(Name))
		Name, err = DecodeK8sResName(Name)
		assert.Nil(t, err)
		assert.Equal(t, name, Name)
	}
	{
		name = "current.name.suffix"
		Name, err := EncodeK8sResName(name)
		assert.Nil(t, err)
		assert.Equal(t, name, Name)
		assert.True(t, k8sNameCheck.MatchString(Name))
		Name, err = DecodeK8sResName(Name)
		assert.Nil(t, err)
		assert.Equal(t, name, Name)
	}
}
