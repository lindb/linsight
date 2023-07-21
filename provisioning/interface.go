// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package provisioning

import (
	depspkg "github.com/lindb/linsight/provisioning/deps"
)

//go:generate mockgen -source=./interface.go -destination=./interface_mock.go -package=provisioning

// NewProviderLoader represents create provider loader function.
type NewProviderLoader func(fileName string, deps *depspkg.ProvisioningDeps) (ProviderLoader, error)

// ProviderLoader represents provisioning provider loader.
type ProviderLoader interface {
	// Load returns provisioning providers from config file.
	Load() []Provider
}

// Provider represents provisioning provider.
type Provider interface {
	// Name returns provider's identification.
	Name() string
	// Run runs provisioning resource discovery.
	Run()
}
