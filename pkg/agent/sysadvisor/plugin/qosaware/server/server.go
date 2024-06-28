/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/metacache"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/plugin/qosaware/resource"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/types"
	"github.com/kubewharf/katalyst-core/pkg/config"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
)

// QRMServer is a wrapper of all qrm plugin servers, which synchronize and merge pod and
// container lifecycle information, resource allocation and provision result with QRM plugins
type QRMServer interface {
	Run(ctx context.Context)
}

// subQRMServer is sub server of qrm server to synchronize information of
// one resource dimension with a specific qrm plugin
type subQRMServer interface {
	Name() string
	Start() error
	Stop() error
	// RegisterAdvisorServer registers resource server and its implementation to the gRPC server.
	RegisterAdvisorServer()
}

type qrmServerWrapper struct {
	serversToRun map[v1.ResourceName]subQRMServer
}

// NewQRMServer returns a qrm server wrapper, which instantiates
// all required qrm plugin servers according to config
func NewQRMServer(advisorWrapper resource.ResourceAdvisor, conf *config.Configuration,
	metaCache metacache.MetaCache, metaServer *metaserver.MetaServer, emitter metrics.MetricEmitter,
) (QRMServer, error) {
	qrmServer := qrmServerWrapper{
		serversToRun: make(map[v1.ResourceName]subQRMServer),
	}

	for _, resourceNameStr := range conf.QRMServers {
		resourceName := v1.ResourceName(resourceNameStr)
		server, err := newSubQRMServer(resourceName, advisorWrapper, conf, metaCache, metaServer, emitter)
		if err != nil {
			return nil, fmt.Errorf("new qrm plugin server for %v failed: %v", resourceName, err)
		} else {
			qrmServer.serversToRun[resourceName] = server
		}
	}

	return &qrmServer, nil
}

func (qs *qrmServerWrapper) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, server := range qs.serversToRun {
		wg.Add(1)
		go func(subQRMServer subQRMServer) {
			defer wg.Done()
			_ = wait.PollImmediateUntil(2*time.Second, func() (done bool, err error) {
				klog.Infof("[qosaware-server] starting %v", subQRMServer.Name())
				if err := subQRMServer.Start(); err != nil {
					klog.Errorf("[qosaware-server] start %v failed: %v", subQRMServer.Name(), err)
					return false, nil
				}
				klog.Infof("[qosaware-server] %v started", subQRMServer.Name())
				return true, nil
			}, ctx.Done())
		}(server)
	}
	wg.Wait()

	<-ctx.Done()

	for _, server := range qs.serversToRun {
		if err := server.Stop(); err != nil {
			klog.Errorf("[qosaware-server] stop %v failed: %v", server.Name(), err)
		}
	}
}

func newSubQRMServer(resourceName v1.ResourceName, advisorWrapper resource.ResourceAdvisor,
	conf *config.Configuration, metaCache metacache.MetaCache, metaServer *metaserver.MetaServer, emitter metrics.MetricEmitter,
) (subQRMServer, error) {
	switch resourceName {
	case v1.ResourceCPU:
		subAdvisor, err := advisorWrapper.GetSubAdvisor(types.QoSResourceCPU)
		if err != nil {
			return nil, err
		}
		advisorRecvChInterface, advisorSendChInterface := subAdvisor.GetChannels()
		advisorRecvCh := advisorRecvChInterface.(chan types.TriggerInfo)
		advisorSendCh := advisorSendChInterface.(chan types.InternalCPUCalculationResult)
		return NewCPUServer(advisorSendCh, advisorRecvCh, conf, metaCache, metaServer, emitter)
	case v1.ResourceMemory:
		subAdvisor, err := advisorWrapper.GetSubAdvisor(types.QoSResourceMemory)
		if err != nil {
			return nil, err
		}
		advisorRecvChInterface, advisorSendChInterface := subAdvisor.GetChannels()
		advisorRecvCh := advisorRecvChInterface.(chan types.TriggerInfo)
		advisorSendCh := advisorSendChInterface.(chan types.InternalMemoryCalculationResult)
		return NewMemoryServer(advisorSendCh, advisorRecvCh, conf, metaCache, metaServer, emitter)
	default:
		return nil, fmt.Errorf("illegal resource %v", resourceName)
	}
}
