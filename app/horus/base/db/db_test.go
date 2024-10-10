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

package db_test

import (
	"github.com/apache/dubbo-kubernetes/app/horus/base/config"
	"github.com/apache/dubbo-kubernetes/app/horus/base/db"
	"testing"
	"time"
)

func TestDataBase(t *testing.T) {
	mc := &config.MysqlConfiguration{
		Address: "root:root@tcp(127.0.0.1:3306)/horus?charset=utf8&parseTime=True",
		Debug:   true,
	}

	err := db.InitDataBase(mc)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
		return
	}

	today := time.Now().Format("2006-01-02 15:04:05")
	data := db.NodeDataInfo{
		NodeName:    "db-test",
		NodeIP:      "1.1.1.1",
		Sn:          "123",
		ClusterName: "beijing01",
		ModuleName:  "model",
		Reason:      "time wrong",
		Restart:     0,
		Repair:      0,
		FirstDate:   today,
	}
	id, err := data.Add()
	t.Logf("test.add id:%v err:%v", id, err)
}
