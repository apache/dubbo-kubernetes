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

package catalog

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"strconv"
)

type Instance struct {
	Id          string `json:"id"`
	Address     string `json:"address"`
	InterCpPort uint16 `json:"interCpPort"`
	Leader      bool   `json:"leader"`
}

func (i Instance) InterCpURL() string {
	return fmt.Sprintf("grpcs://%s", net.JoinHostPort(i.Address, strconv.Itoa(int(i.InterCpPort))))
}

type Reader interface {
	Instances(context.Context) ([]Instance, error)
}

type Catalog interface {
	Reader
	Replace(context.Context, []Instance) (bool, error)
	ReplaceLeader(context.Context, Instance) error
}

var (
	ErrNoLeader         = errors.New("leader not found")
	ErrInstanceNotFound = errors.New("instance not found")
)

func Leader(ctx context.Context, catalog Catalog) (Instance, error) {
	instances, err := catalog.Instances(ctx)
	if err != nil {
		return Instance{}, err
	}
	for _, instance := range instances {
		if instance.Leader {
			return instance, nil
		}
	}
	return Instance{}, ErrNoLeader
}

func InstanceOfID(ctx context.Context, catalog Catalog, id string) (Instance, error) {
	instances, err := catalog.Instances(ctx)
	if err != nil {
		return Instance{}, err
	}
	for _, instance := range instances {
		if instance.Id == id {
			return instance, nil
		}
	}
	return Instance{}, ErrInstanceNotFound
}

type InstancesByID []Instance

func (a InstancesByID) Len() int      { return len(a) }
func (a InstancesByID) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a InstancesByID) Less(i, j int) bool {
	return a[i].Id < a[j].Id
}
