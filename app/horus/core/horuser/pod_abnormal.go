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

package horuser

import (
	"context"
	"k8s.io/apimachinery/pkg/util/wait"
	"sync"
	"time"
)

func (h *Horuser) PodAbnormalCleanManager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, h.PodAbnormalClean, time.Duration(h.cc.PodAbnormal.IntervalSecond)*time.Second)
	<-ctx.Done()
	return nil
}

func (h *Horuser) PodAbnormalClean(ctx context.Context) {
	var wg sync.WaitGroup
	for cn := range h.cc.PodAbnormal.KubeMultiple {
		wg.Add(1)
		go func(clusterName string) {
			defer wg.Done()
		}(cn)
	}
	wg.Wait()
}
