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
	"context"
	"flag"
	"github.com/apache/dubbo-kubernetes/app/horus/basic/config"
	"github.com/apache/dubbo-kubernetes/app/horus/core/db"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	address    string
	configFile string
)

func main() {
	flag.StringVar(&configFile, "configFile", "deploy/horus/horus.yaml", "horus config file")
	flag.StringVar(&address, "address", "0.0.0.0:38089", "horus address")
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
	group, stopChan := setupStopChanWithContext()
	ctx, cancel := context.WithCancel(context.Background())
	group.Add(func() error {
		for {
			select {
			case <-stopChan:
				cancel()
				return nil
			case <-ctx.Done():
				cancel()
				return nil
			}
		}
	})
	group.Add(func() error {
		http.Handle("/metrics", promhttp.Handler())
		srv := http.Server{Addr: address}
		err := srv.ListenAndServe()
		if err != nil {
			klog.Errorf("horus metrics err:%v", err)
		}
		return nil
	})
	group.Wait()
}

type WaitGroup struct {
	wg sync.WaitGroup
}

func (g *WaitGroup) Add(f func() error) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		err := f()
		if err != nil {
			return
		}
	}()
}

func (g *WaitGroup) Wait() {
	g.wg.Wait()
}

func setupStopChanWithContext() (*WaitGroup, <-chan struct{}) {
	stopChan := make(chan struct{})
	SignalChan := make(chan os.Signal, 1)
	signal.Notify(SignalChan, syscall.SIGTERM, syscall.SIGQUIT)
	g := WaitGroup{}
	g.Add(func() error {
		<-stopChan
		close(stopChan)
		return nil
	})
	return &g, stopChan
}
