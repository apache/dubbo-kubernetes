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
	"github.com/apache/dubbo-kubernetes/app/horus/base/config"
	"github.com/apache/dubbo-kubernetes/app/horus/base/db"
	"github.com/apache/dubbo-kubernetes/app/horus/core/horuser"
	"github.com/apache/dubbo-kubernetes/app/horus/core/ticker"
	"github.com/prometheus/client_golang/prometheus"
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
	flag.StringVar(&configFile, "configFile", "../../manifests/horus/horus.yaml", "horus config file")
	flag.StringVar(&address, "address", "0.0.0.0:38089", "horus address")
	klog.InitFlags(flag.CommandLine)
	flag.Parse()

	c, err := config.LoadFile(configFile)
	if err != nil {
		klog.Errorf("load config file failed err:%+v", err)
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
	horus := horuser.NewHoruser(c)
	prometheus.MustRegister(horus)
	group, stopChan := setupStopChanWithContext()
	ctx, cancel := context.WithCancel(context.Background())
	group.Add(func() error {
		for {
			select {
			case <-stopChan:
				cancel()
				return nil
			case <-ctx.Done():
				return nil
			}
		}
	})
	group.Add(func() error {
		http.Handle("/metrics", promhttp.Handler())
		srv := http.Server{Addr: c.Address}
		err := srv.ListenAndServe()
		if err != nil {
			klog.Errorf("horus metrics err:%v", err)
			return err
		}
		return nil
	})
	group.Add(func() error {
		klog.Info("horus ticker manager start success.")
		err := ticker.Manager(ctx)
		if err != nil {
			klog.Errorf("horus ticker manager start failed err:%v", err)
			return err
		}
		return nil
	})
	group.Add(func() error {
		klog.Info("horus node recovery manager start success.")
		err := horus.RecoveryManager(ctx)
		if err != nil {
			klog.Errorf("horus node recovery manager start failed err:%v", err)
			return err
		}
		return nil
	})
	group.Add(func() error {
		if c.CustomModular.Enabled {
			klog.Info("horus node customize modular manager start success.")
			err := horus.CustomizeModularManager(ctx)
			if err != nil {
				klog.Errorf("horus node customize modular manager start failed err:%v", err)
				return err
			}
		}
		return nil
	})
	group.Add(func() error {
		if c.NodeDownTime.Enabled {
			klog.Info("horus node downtime manager start success.")
			err := horus.DownTimeManager(ctx)
			if err != nil {
				klog.Errorf("horus node downtime manager start failed err:%v", err)
				return err
			}
		}
		return nil
	})
	group.Add(func() error {
		klog.Info("horus node downtime restart manager start success.")
		err := horus.DowntimeRestartManager(ctx)
		if err != nil {
			klog.Errorf("horus node downtime restart manager start failed err:%v", err)
			return err
		}
		return nil
	})
	group.Add(func() error {
		if c.PodStagnationCleaner.Enabled {
			klog.Info("horus pod stagnation clean manager start success.")
			err := horus.PodStagnationCleanManager(ctx)
			if err != nil {
				klog.Errorf("horus pod stagnation clean manager start failed err:%v", err)
				return err
			}
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
		_ = f()
	}()
}

func (g *WaitGroup) Wait() {
	g.wg.Wait()
}

func setupStopChanWithContext() (*WaitGroup, <-chan struct{}) {
	stopChan := make(chan struct{})
	SignalChan := make(chan os.Signal, 1)
	signal.Notify(SignalChan, syscall.SIGTERM, syscall.SIGQUIT)
	g := &WaitGroup{}
	g.Add(func() error {
		select {
		case <-SignalChan:
			close(SignalChan)
		}
		return nil
	})
	return g, stopChan
}
