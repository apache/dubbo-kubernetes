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

package db

import (
	"fmt"
	"github.com/apache/dubbo-kubernetes/app/horus/basic/config"
	_ "github.com/go-sql-driver/mysql"
	"time"
	"xorm.io/xorm"
	xlog "xorm.io/xorm/log"
)

type NodeDataInfo struct {
	Id              int64  `json:"id"`
	NodeName        string `json:"nodeName"`
	NodeIP          string `json:"nodeIP"`
	Sn              string `json:"sn"`
	ClusterName     string `json:"clusterName"`
	ModuleName      string `json:"moduleName"`
	Reason          string `json:"reason"`
	Restart         uint32 `json:"restart"`
	Repair          uint32 `json:"repair"`
	RepairTicketUrl string `json:"repairTicketUrl"`
	FirstDate       string `json:"firstDate"`
	LastDate        string `json:"lastDate"`
	CreateTime      string `json:"createTime" xorm:"createTime created"`
	UpdateTime      string `json:"updateTime" xorm:"updateTime updated"`
}

type PodDataInfo struct {
	Id              uint32 `json:"id"`
	PodName         string `json:"podName"`
	PodIP           string `json:"podIP"`
	Sn              string `json:"sn"`
	NodeName        string `json:"nodeName"`
	ClusterName     string `json:"clusterName"`
	ModuleName      string `json:"moduleName"`
	Reason          string `json:"reason"`
	Restart         int32  `json:"restart"`
	Repair          int32  `json:"repair"`
	RepairTicketUrl string `json:"repairTicketUrl"`
	FirstDate       string `json:"firstDate"`
	LastDate        string `json:"lastDate"`
	CreateTime      string `json:"createTime"`
	UpdateTime      string `json:"updateTime"`
}

var (
	db *xorm.Engine
)

func InitDataBase(mc *config.MysqlConfiguration) error {
	data, err := xorm.NewEngine("mysql", mc.Addr)
	if err != nil {
		fmt.Printf("Unable to connect to mysql server:\n  addr: %s\n  err: %v\n", mc.Addr, err)
	}
	data.Logger().SetLevel(xlog.LOG_INFO)
	data.ShowSQL(mc.Debug)
	db = data
	return nil
}

func (n *NodeDataInfo) Add() (int64, error) {
	exist, err := db.Exist(n)
	if err != nil {
		return 0, err
	}
	if exist {
		return n.Id, nil
	}

	affected, err := db.Insert(n)
	if err != nil {
		return 0, err
	}

	if affected > 0 {
		return n.Id, nil
	}

	return 0, fmt.Errorf("failed to insert record")
}

func (n *NodeDataInfo) Get() (*NodeDataInfo, error) {
	exist, err := db.Get(n)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return n, nil
}

func (n *NodeDataInfo) Update() (bool, error) {
	firstDate := time.Now().Format("2006-01-02 15:04:05")
	n.FirstDate = firstDate

	row, err := db.Where(fmt.Sprintf("id=%d", n.Id)).Update(n)
	if err != nil {
		return false, err
	}
	if row > 0 {
		return true, nil
	}
	return false, nil
}

func (n *NodeDataInfo) Check() (bool, error) {
	exist, err := db.Exist(n)
	return exist, err
}
