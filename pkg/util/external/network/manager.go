// Copyright 2022 The Katalyst Authors.
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

package network

import (
	"errors"

	"github.com/kubewharf/katalyst-core/pkg/util/cgroup/common"
)

const (
	// SubManagerName is the name of NetworkManager.
	SubManagerName = "network"
)

// NetworkManager provides methods that control network resources.
type NetworkManager interface {
	ApplyNetClass(podUID, containerId string, data *common.NetClsData) error
	ClearNetClass(cgroupID uint64) error
}

type defaultNetworkManager struct{}

// NewDefaultManager returns a defaultNetworkManager.
func NewDefaultManager() NetworkManager {
	return &defaultNetworkManager{}
}

// ApplyNetClass applies the net class config for a container.
func (*defaultNetworkManager) ApplyNetClass(podUID, containerId string, data *common.NetClsData) error {
	// TODO: implement traffic tagging by using eBPF
	return errors.New("not implemented yet")
}

// ClearNetClass clears the net class config for a container.
func (*defaultNetworkManager) ClearNetClass(cgroupID uint64) error {
	// TODO: clear the eBPF map when a pod is removed
	return errors.New("not implemented yet")
}
