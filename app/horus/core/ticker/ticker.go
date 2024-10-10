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

package ticker

import (
	"context"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"
)

func Manager(ctx context.Context) error {
	go wait.UntilWithContext(ctx, tickerFunc, 30*time.Second)
	<-ctx.Done()
	klog.Infof("ticker Manager exit receive quit signal")
	return nil
}

func tickerFunc(ctx context.Context) {
	klog.V(2).Infof("tickerFunc Running")
}
