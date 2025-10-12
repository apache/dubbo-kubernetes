package nacos

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

// Config holds the configuration for connecting to Nacos registry
type Config struct {
	// ServerAddrs is a list of Nacos server addresses
	ServerAddrs []string `json:"serverAddrs" yaml:"serverAddrs"`

	// Namespace is the Nacos namespace to use
	Namespace string `json:"namespace" yaml:"namespace"`

	// Group is the service group name
	Group string `json:"group" yaml:"group"`

	// Username for authentication
	Username string `json:"username" yaml:"username"`

	// Password for authentication
	Password string `json:"password" yaml:"password"`

	// AccessKey for authentication
	AccessKey string `json:"accessKey" yaml:"accessKey"`

	// SecretKey for authentication
	SecretKey string `json:"secretKey" yaml:"secretKey"`

	// Timeout for connecting to Nacos
	Timeout time.Duration `json:"timeout" yaml:"timeout"`

	// RetryInterval for reconnecting
	RetryInterval time.Duration `json:"retryInterval" yaml:"retryInterval"`

	// MaxRetries for connection attempts
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`
}

// DefaultConfig returns a default Nacos configuration
func DefaultConfig() *Config {
	return &Config{
		ServerAddrs:   []string{"127.0.0.1:8848"},
		Namespace:     "public",
		Group:         "DEFAULT_GROUP",
		Timeout:       5 * time.Second,
		RetryInterval: 30 * time.Second,
		MaxRetries:    3,
	}
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	if len(c.ServerAddrs) == 0 {
		return fmt.Errorf("nacos server addresses cannot be empty")
	}
	if c.Namespace == "" {
		c.Namespace = "public"
	}
	if c.Group == "" {
		c.Group = "DEFAULT_GROUP"
	}
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second
	}
	return nil
}
