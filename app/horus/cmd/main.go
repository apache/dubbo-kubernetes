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

package main

import (
	"flag"
	"github.com/apache/dubbo-kubernetes/app/horus/basic/config"
	"github.com/apache/dubbo-kubernetes/app/horus/core/db"
	"k8s.io/klog/v2"
)

var (
	httpAddr   string
	configFile string
)

func main() {
	flag.StringVar(&configFile, "configFile", "deploy/horus/horus.yaml", "horus config file")
	flag.StringVar(&httpAddr, "httpAddr", "0.0.0.0:38089", "horus http addr")
	klog.InitFlags(flag.CommandLine)
	flag.Parse()

	c, err := config.LoadFile(configFile)
	if err != nil {
		klog.Errorf("load config file failed err:%+v", c)
		return
	} else {
		klog.Infof("load config file success.")
	}

	err = db.InitDataBase(c.Mysql)
	if err != nil {
		klog.Errorf("horus db initial failed err:%v", err)
		return
	} else {
		klog.Infof("horus db initial success.")
	}

}