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

package traditional

import "testing"

func TestSplitAppAndRevision(t *testing.T) {
	name := "dubbo-springboot-demo-lixinyang-bdc0958191bba7a0f050a32709ee1111"
	app, revision := splitAppAndRevision(name)
	if app != "dubbo-springboot-demo-lixinyang" && revision != "bdc0958191bba7a0f050a32709ee1111" {
		t.Error("解析错误")
	}
}
