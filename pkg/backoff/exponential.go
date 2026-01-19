//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backoff

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	defaultInitialInterval = 500 * time.Millisecond
	defaultMaxInterval     = 60 * time.Second
)

// BackOff is a backoff policy for retrying an operation.
type BackOff interface {
	NextBackOff() time.Duration
	Reset()
	RetryWithContext(ctx context.Context, operation func() error) error
}

type Option struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
}

func DefaultOption() Option {
	return Option{
		InitialInterval: defaultInitialInterval,
		MaxInterval:     defaultMaxInterval,
	}
}

type ExponentialBackOff struct {
	exponentialBackOff *backoff.ExponentialBackOff
}

func NewExponentialBackOff(o Option) BackOff {
	b := ExponentialBackOff{}
	b.exponentialBackOff = backoff.NewExponentialBackOff()
	b.exponentialBackOff.InitialInterval = o.InitialInterval
	b.exponentialBackOff.MaxInterval = o.MaxInterval
	b.Reset()
	return b
}

func (b ExponentialBackOff) NextBackOff() time.Duration {
	duration := b.exponentialBackOff.NextBackOff()
	if duration == b.exponentialBackOff.Stop {
		return b.exponentialBackOff.MaxInterval
	}
	return duration
}

func (b ExponentialBackOff) Reset() {
	b.exponentialBackOff.Reset()
}

func (b ExponentialBackOff) RetryWithContext(ctx context.Context, operation func() error) error {
	b.Reset()
	for {
		err := operation()
		if err == nil {
			return nil
		}
		next := b.NextBackOff()
		select {
		case <-ctx.Done():
			return fmt.Errorf("%v with last error: %v", context.DeadlineExceeded, err)
		case <-time.After(next):
		}
	}
}
