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

package store

import (
	"encoding/json"
	"net/url"
	"strings"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"

	"github.com/kelseyhightower/envconfig"

	"github.com/pkg/errors"

	"go.uber.org/multierr"
)

import (
	"github.com/apache/dubbo-kubernetes/pkg/config"
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/resources/k8s"
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/resources/mysql"
	"github.com/apache/dubbo-kubernetes/pkg/config/plugins/resources/zookeeper"
	config_types "github.com/apache/dubbo-kubernetes/pkg/config/types"
)

var _ config.Config = &StoreConfig{}

type StoreType = string

const (
	KubernetesStore StoreType = "kubernetes"
	MemoryStore     StoreType = "memory"
	MyStore         StoreType = "mysql"
	Traditional     StoreType = "traditional"
)

// StoreConfig defines Resource Store configuration
type StoreConfig struct {
	// Type of Store used in the Control Plane. Can be either "kubernetes", "postgres" or "memory"
	Type StoreType `json:"type" envconfig:"DUBBO_STORE_TYPE"`
	// Kubernetes Store configuration
	Kubernetes *k8s.KubernetesStoreConfig `json:"kubernetes"`
	// Zookeeper Store configuration
	Zookeeper *zookeeper.ZookeeperStoreConfig `json:"zookeeper"`
	// Mysql Store configuration
	Mysql       *mysql.MysqlStoreConfig `json:"mysql"`
	Traditional Registry                `json:"traditional" envconfig:"DUBBO_STORE_TRADITIONAL_REGISTRY"`
	// Cache configuration
	Cache CacheStoreConfig `json:"cache"`
	// Upsert configuration
	Upsert UpsertConfig `json:"upsert"`
	// UnsafeDelete skips validation of resource delete.
	// For example you don't have to delete all Dataplane objects before you delete a Mesh
	UnsafeDelete bool `json:"unsafeDelete" envconfig:"DUBBO_STORE_UNSAFE_DELETE"`
}

func DefaultStoreConfig() *StoreConfig {
	return &StoreConfig{
		Type:        KubernetesStore,
		Kubernetes:  k8s.DefaultKubernetesStoreConfig(),
		Cache:       DefaultCacheStoreConfig(),
		Upsert:      DefaultUpsertConfig(),
		Mysql:       DefaultMysqlConfig(),
		Traditional: DefaultTraditionalConfig(),
	}
}

func (s *StoreConfig) Sanitize() {
	s.Kubernetes.Sanitize()
	s.Cache.Sanitize()
}

func (s *StoreConfig) PostProcess() error {
	return multierr.Combine(
		s.Kubernetes.PostProcess(),
		s.Cache.PostProcess(),
	)
}

func (s *StoreConfig) Validate() error {
	switch s.Type {
	case KubernetesStore:
		if err := s.Kubernetes.Validate(); err != nil {
			return errors.Wrap(err, "Kubernetes validation failed")
		}
	case MemoryStore:
		return nil
	case MyStore:

	default:
		return errors.Errorf("Type should be either %s or %s", KubernetesStore, MemoryStore)
	}
	if err := s.Cache.Validate(); err != nil {
		return errors.Wrap(err, "Cache validation failed")
	}
	return nil
}

var _ config.Config = &CacheStoreConfig{}

type CacheStoreConfig struct {
	config.BaseConfig

	Enabled        bool                  `json:"enabled" envconfig:"DUBBO_STORE_CACHE_ENABLED"`
	ExpirationTime config_types.Duration `json:"expirationTime" envconfig:"DUBBO_STORE_CACHE_EXPIRATION_TIME"`
}

func DefaultCacheStoreConfig() CacheStoreConfig {
	return CacheStoreConfig{
		Enabled:        true,
		ExpirationTime: config_types.Duration{Duration: time.Second},
	}
}

func DefaultUpsertConfig() UpsertConfig {
	return UpsertConfig{
		ConflictRetryBaseBackoff:   config_types.Duration{Duration: 200 * time.Millisecond},
		ConflictRetryMaxTimes:      10,
		ConflictRetryJitterPercent: 30,
	}
}

func DefaultMysqlConfig() *mysql.MysqlStoreConfig {
	return &mysql.MysqlStoreConfig{
		MysqlDsn: "127.0.0.1:6379",
	}
}

func DefaultTraditionalConfig() Registry {
	return Registry{
		ConfigCenter: "zookeeper://127.0.0.1:2181",
		Registry: AddressConfig{
			Address: "zookeeper://127.0.0.1:2181",
		},
		MetadataReport: AddressConfig{
			Address: "zookeeper://127.0.0.1:2181",
		},
	}
}

type UpsertConfig struct {
	config.BaseConfig

	// Base time for exponential backoff on upsert (get and update) operations when retry is enabled
	ConflictRetryBaseBackoff config_types.Duration `json:"conflictRetryBaseBackoff" envconfig:"DUBBO_STORE_UPSERT_CONFLICT_RETRY_BASE_BACKOFF"`
	// Max retries on upsert (get and update) operation when retry is enabled
	ConflictRetryMaxTimes uint `json:"conflictRetryMaxTimes" envconfig:"DUBBO_STORE_UPSERT_CONFLICT_RETRY_MAX_TIMES"`
	// Percentage of jitter. For example: if backoff is 20s, and this value 10, the backoff will be between 18s and 22s.
	ConflictRetryJitterPercent uint `json:"conflictRetryJitterPercent" envconfig:"DUBBO_STORE_UPSERT_CONFLICT_RETRY_JITTER_PERCENT"`
}

func (u *UpsertConfig) Validate() error {
	if u.ConflictRetryBaseBackoff.Duration < 0 {
		return errors.New("RetryBaseBackoff cannot be lower than 0")
	}
	return nil
}

var _ config.Config = &UpsertConfig{}

type Registry struct {
	ConfigCenter   string        `json:"config_center,omitempty"`
	MetadataReport AddressConfig `json:"metadata_report,omitempty"`
	Registry       AddressConfig `json:"registry,omitempty"`
}

func (r *Registry) Sanitize() {}

func (r *Registry) Validate() error {
	return nil
}

func (r *Registry) PostProcess() error {
	return nil
}

func (r *Registry) Decode(val string) error {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "{") {
		// try to unmarshal as json
		return json.Unmarshal([]byte(val), r)
	}
	// else treat as a global address for all
	r.ConfigCenter = val
	r.MetadataReport.Address = val
	r.Registry.Address = val
	return nil
}

var _ config.Config = &Registry{}

var _ envconfig.Decoder = &Registry{}

type AddressConfig struct {
	Address string   `json:"address,omitempty"`
	Url     *url.URL `json:"-"`
}

func (a *AddressConfig) Sanitize() {}

var _ config.Config = &AddressConfig{}

func (a *AddressConfig) PostProcess() error {
	return nil
}

func (a *AddressConfig) Validate() error {
	return nil
}

func (c *AddressConfig) GetProtocol() string {
	return c.Url.Scheme
}

func (c *AddressConfig) GetAddress() string {
	return c.Url.Host
}

func (c *AddressConfig) GetUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.ConfigNamespaceKey, c.param("namespace", ""))
	urlMap.Set(constant.ConfigGroupKey, c.param(constant.GroupKey, "dubbo"))
	urlMap.Set(constant.MetadataReportGroupKey, c.param(constant.GroupKey, "dubbo"))
	urlMap.Set(constant.ClientNameKey, clientNameID(c.Url.Scheme, c.Url.Host))
	return urlMap
}

func (c *AddressConfig) param(key string, defaultValue string) string {
	param := c.Url.Query().Get(key)
	if len(param) > 0 {
		return param
	}
	return defaultValue
}

func (c *AddressConfig) ToURL() (*common.URL, error) {
	return common.NewURL(c.GetAddress(),
		common.WithProtocol(c.GetProtocol()),
		common.WithParams(c.GetUrlMap()),
		common.WithParamsValue("registry", c.GetProtocol()),
		common.WithUsername(c.param("username", "")),
		common.WithPassword(c.param("password", "")),
	)
}

func clientNameID(protocol, address string) string {
	return strings.Join([]string{protocol, address}, "-")
}
