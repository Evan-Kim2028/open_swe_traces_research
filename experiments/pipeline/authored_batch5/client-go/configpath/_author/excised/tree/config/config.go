// Copyright 2021 KVStore Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// NOTE: The code in this file is based on code from the
// SQLEngine project, licensed under the Apache License v 2.0
//
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/config/config.go
//

// Copyright 2021 Acme, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	_ "fmt"
	_ "net/url"
	_ "strings"
	"sync/atomic"

	_ "example.internal/kvstore/v2/internal/logutil"
	_ "example.internal/kvstore/v2/oracle"
	_ "example.internal/kvstore/v2/util"
	_ "github.com/pkg/errors"
	_ "go.uber.org/zap"
)

var (
	globalConf atomic.Value
)

const (
	// DefStoresRefreshInterval is the default value of StoresRefreshInterval
	DefStoresRefreshInterval = 60
)

func init() {
	conf := DefaultConfig()
	StoreGlobalConfig(&conf)
}

// Config contains configuration options.
type Config struct {
	CommitterConcurrency int
	MaxTxnTTL            uint64
	TiKVClient           TiKVClient
	Security             Security
	PDClient             PDClient
	PessimisticTxn       PessimisticTxn
	TxnLocalLatches      TxnLocalLatches
	// StoresRefreshInterval indicates the interval of refreshing stores info, the unit is second.
	StoresRefreshInterval uint64
	OpenTracingEnable     bool
	Path                  string
	EnableForwarding      bool
	TxnScope              string
	EnableAsyncCommit     bool
	Enable1PC             bool
	// RegionsRefreshInterval indicates the interval of loading regions info, the unit is second, if RegionsRefreshInterval == 0, it will be disabled.
	RegionsRefreshInterval uint64
	// EnablePreload indicates whether to preload region info when initializing the client.
	EnablePreload bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		CommitterConcurrency:  128,
		MaxTxnTTL:             60 * 60 * 1000, // 1hour
		TiKVClient:            DefaultTiKVClient(),
		PDClient:              DefaultPDClient(),
		TxnLocalLatches:       DefaultTxnLocalLatches(),
		StoresRefreshInterval: DefStoresRefreshInterval,
		OpenTracingEnable:     false,
		Path:                  "",
		EnableForwarding:      false,
		TxnScope:              "",
		EnableAsyncCommit:     false,
		Enable1PC:             false,
	}
}

// PDClient is the config for Meta client.
type PDClient struct {
	// PDServerTimeout is the max time which Meta client will wait for the Meta server in seconds.
	PDServerTimeout uint `toml:"pd-server-timeout" json:"pd-server-timeout"`
}

// DefaultPDClient returns the default configuration for PDClient
func DefaultPDClient() PDClient {
	return PDClient{
		PDServerTimeout: 3,
	}
}

// TxnLocalLatches is the TxnLocalLatches section of the config.
type TxnLocalLatches struct {
	Enabled  bool `toml:"-" json:"-"`
	Capacity uint `toml:"-" json:"-"`
}

// DefaultTxnLocalLatches returns the default configuration for TxnLocalLatches
func DefaultTxnLocalLatches() TxnLocalLatches {
	return TxnLocalLatches{
		Enabled:  false,
		Capacity: 0,
	}
}

// Valid returns true if the configuration is valid.
func (c *TxnLocalLatches) Valid() error {
	panic("excised: TxnLocalLatches.Valid")
}

// PessimisticTxn is the config for pessimistic transaction.
type PessimisticTxn struct {
	// The max count of retry for a single statement in a pessimistic transaction.
	MaxRetryCount uint `toml:"max-retry-count" json:"max-retry-count"`
}

// GetGlobalConfig returns the global configuration for this server.
// It should store configuration from command line and configuration file.
// Other parts of the system can read the global configuration use this function.
func GetGlobalConfig() *Config {
	return globalConf.Load().(*Config)
}

// StoreGlobalConfig stores a new config to the globalConf. It mostly uses in the test to avoid some data races.
func StoreGlobalConfig(config *Config) {
	globalConf.Store(config)
}

// UpdateGlobal updates the global config, and provide a restore function that can be used to restore to the original.
func UpdateGlobal(f func(conf *Config)) func() {
	g := GetGlobalConfig()
	restore := func() {
		StoreGlobalConfig(g)
	}
	newConf := *g
	f(&newConf)
	StoreGlobalConfig(&newConf)
	return restore
}

// GetTxnScopeFromConfig extracts @@txn_scope value from the config.
func GetTxnScopeFromConfig() string {
	panic("excised: GetTxnScopeFromConfig")
}

// ParsePath parses this path.
// Path example: kvstore://etcd-node1:port,etcd-node2:port?cluster=1&disableGC=false&keyspaceName=SomeKeyspace
func ParsePath(path string) (etcdAddrs []string, disableGC bool, keyspaceName string, err error) {
	panic("excised: ParsePath")
}
