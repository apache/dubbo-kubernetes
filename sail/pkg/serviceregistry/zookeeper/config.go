package zookeeper

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

import (
	"fmt"
	"time"
)

// Config holds the configuration for connecting to Zookeeper registry
type Config struct {
	// Servers is a list of Zookeeper server addresses
	Servers []string `json:"servers" yaml:"servers"`

	// Root is the root path for Dubbo services in Zookeeper
	Root string `json:"root" yaml:"root"`

	// Timeout for connecting to Zookeeper
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// SessionTimeout for Zookeeper session
	SessionTimeout time.Duration `json:"sessionTimeout" yaml:"sessionTimeout"`

	// RetryInterval for reconnecting
	RetryInterval time.Duration `json:"retryInterval" yaml:"retryInterval"`

	// MaxRetries for connection attempts
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`

	// Username for authentication (if required)
	Username string `json:"username" yaml:"username"`

	// Password for authentication (if required)
	Password string `json:"password" yaml:"password"`
}

// DefaultConfig returns a default Zookeeper configuration
func DefaultConfig() *Config {
	return &Config{
		Servers:        []string{"127.0.0.1:2181"},
		Root:           "/dubbo",
		Timeout:        5 * time.Second,
		SessionTimeout: 60 * time.Second,
		RetryInterval:  30 * time.Second,
		MaxRetries:     3,
	}
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("zookeeper servers cannot be empty")
	}
	if c.Root == "" {
		c.Root = "/dubbo"
	}
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second
	}
	if c.SessionTimeout <= 0 {
		c.SessionTimeout = 60 * time.Second
	}
	return nil
}
