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

package admin

import (
	"github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
)

type Admin struct {
	AdminPort      int           `json:"Port"`
	ConfigCenter   string        `json:"configCenter"`
	MetadataReport AddressConfig `json:"metadataReport"`
	Registry       AddressConfig `json:"registry"`
	Prometheus     Prometheus    `json:"prometheus"`
	Grafana        Grafana       `json:"grafana"`
	MysqlDSN       string        `json:"mysqlDSN"`
}

type Prometheus struct {
	Address     string `json:"address"`
	MonitorPort string `json:"monitorPort"`
}

func (c *Prometheus) Sanitize() {}

func (c *Prometheus) Validate() error {
	// TODO Validate admin
	return nil
}

type Grafana struct {
	Address string `json:"address"`
}

func (g *Grafana) Sanitize() {}

func (g *Grafana) Validate() error {
	// TODO Validate admin
	return nil
}

func (c *Admin) Sanitize() {
	c.Prometheus.Sanitize()
	c.Registry.Sanitize()
	c.MetadataReport.Sanitize()
	c.MysqlDSN = config.SanitizedValue
}

func (c *Admin) Validate() error {
	err := c.Prometheus.Validate()
	if err != nil {
		return errors.Wrap(err, "Prometheus validation failed")
	}
	err = c.Registry.Validate()
	if err != nil {
		return errors.Wrap(err, "Registry validation failed")
	}
	err = c.MetadataReport.Validate()
	if err != nil {
		return errors.Wrap(err, "MetadataReport validation failed")
	}
	return nil
}
