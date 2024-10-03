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
	"github.com/apache/dubbo-kubernetes/app/horus/base/config"
	_ "github.com/go-sql-driver/mysql"
	"time"
	"xorm.io/xorm"
	xlog "xorm.io/xorm/log"
)

type NodeDataInfo struct {
	Id              int64  `json:"id"`
	NodeName        string `json:"node_name" xorm:"node_name"`
	NodeIP          string `json:"node_ip" xorm:"node_ip"`
	Sn              string `json:"sn"`
	ClusterName     string `json:"cluster_name" xorm:"cluster_name"`
	ModuleName      string `json:"module_name" xorm:"module_name"`
	Reason          string `json:"reason"`
	Restart         uint32 `json:"restart"`
	Repair          uint32 `json:"repair"`
	RepairTicketUrl string `json:"repair_ticket_url" xorm:"repair_ticket_url"`
	FirstDate       string `json:"first_date" xorm:"first_date"`
	CreateTime      string `json:"create_time" xorm:"create_time created"`
	UpdateTime      string `json:"update_time" xorm:"update_time updated"`
	RecoveryMark    int64  `json:"recovery_mark" xorm:"recovery_mark"`
	RecoveryQL      string `json:"recovery_ql" xorm:"recovery_ql"`
}

type PodDataInfo struct {
	Id              int64  `json:"id"`
	PodName         string `json:"pod_name" xorm:"pod_name"`
	PodIP           string `json:"pod_ip" xorm:"pod_ip"`
	Sn              string `json:"sn"`
	NodeName        string `json:"node_name" xorm:"node_name"`
	ClusterName     string `json:"cluster_name" xorm:"cluster_name"`
	ModuleName      string `json:"module_name" xorm:"module_name"`
	Reason          string `json:"reason"`
	Restart         int32  `json:"restart"`
	Repair          int32  `json:"repair"`
	RepairTicketUrl string `json:"repair_ticket_url" xorm:"repair_ticket_url"`
	FirstDate       string `json:"first_date" xorm:"first_date"`
	CreateTime      string `json:"create_time" xorm:"create_time created"`
	UpdateTime      string `json:"update_time" xorm:"update_time updated"`
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
	row, err := db.Insert(n)
	return row, err
}

func (p *PodDataInfo) Add() (int64, error) {
	row, err := db.Insert(p)
	return row, err
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

func (p *PodDataInfo) Get() (*PodDataInfo, error) {
	exist, err := db.Get(p)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, nil
	}
	return p, nil
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

func (p *PodDataInfo) Update() (bool, error) {
	firstDate := time.Now().Format("2006-01-02 15:04:05")
	p.FirstDate = firstDate

	row, err := db.Where(fmt.Sprintf("id=%d", p.Id)).Update(p)
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

func (p *PodDataInfo) Check() (bool, error) {
	exist, err := db.Exist(p)
	return exist, err
}

func (n *NodeDataInfo) AddOrGet() (int64, error) {
	exist, _ := n.Check()
	if exist {
		return n.Id, nil
	}
	row, err := n.Add()
	return row, err
}

func (p *PodDataInfo) AddOrGet() (int64, error) {
	exist, _ := p.Check()
	if exist {
		return p.Id, nil
	}
	row, err := p.Add()
	return row, err
}

func GetRecoveryNodeDataInfoDate(day int) ([]NodeDataInfo, error) {
	var ndi []NodeDataInfo
	session := db.Where(fmt.Sprintf("recovery_mark = 0 AND first_date > DATE_SUB(CURDATE(), INTERVAL %d DAY)", day))
	err := session.Find(&ndi)
	return ndi, err
}

func GetRestartNodeDataInfoDate() ([]NodeDataInfo, error) {
	var ndi []NodeDataInfo
	session := db.Where("restart = 0 and repair = 0 and module_name = ?", "node_down")
	err := session.Find(&ndi)
	return ndi, err
}

func GetDailyLimitNodeDataInfoDate(day, module, cluster string) ([]NodeDataInfo, error) {
	var ndi []NodeDataInfo
	session := db.Where("DATE(first_date) = ? AND module_name = ? AND cluster_name = ?", day, module, cluster)
	err := session.Find(&ndi)
	return ndi, err
}

func (n *NodeDataInfo) RecoveryMarker() (bool, error) {
	n.RecoveryMark = 1
	return n.Update()
}

func (n *NodeDataInfo) RestartMarker() (bool, error) {
	n.Restart = 1
	return n.Update()
}

func GetPod() ([]PodDataInfo, error) {
	var pdi []PodDataInfo
	session := db.Where(fmt.Sprintf("id>%d", 0))
	err := session.Find(&pdi)
	return pdi, err
}

func GetPodByName(podName, moduleName string) (*PodDataInfo, error) {
	pdi := PodDataInfo{
		PodName:    podName,
		ModuleName: moduleName,
	}
	return pdi.Get()
}

func GetLimitPodDataInfo(limit, offset int, orderColumn, where string, args ...interface{}) ([]PodDataInfo, error) {
	var pdi []PodDataInfo
	err := db.Where(where, args...).Limit(limit, offset).Desc(orderColumn).Find(pdi)
	if err != nil {
		return nil, err
	}
	return pdi, nil
}
