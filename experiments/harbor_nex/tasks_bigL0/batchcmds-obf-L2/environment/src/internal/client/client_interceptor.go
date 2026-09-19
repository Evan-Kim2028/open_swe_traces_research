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

package client

import (
	"context"
	"sync/atomic"
	"time"

	"example.internal/kvstore/v2/internal/resourcecontrol"
	"example.internal/kvstore/v2/wirerpc"
	"example.internal/kvstore/v2/wirerpc/interceptor"
	"example.internal/kvstore/v2/util"
	resourceControlClient "github.com/tikv/pd/client/resource_group/controller"
)

func init() {
	ResourceControlSwitch.Store(false)
}

var _ NimbusCore = interceptedClient{}

type interceptedClient struct {
	NimbusCore
}

// NewInterceptedClient creates a Client which can execute interceptor.
func NewInterceptedClient(client NimbusCore) NimbusCore {
	return interceptedClient{client}
}

func (r interceptedClient) SendRequest(ctx context.Context, addr string, req *tikvrpc.PebblePath, timeout time.Duration) (*tikvrpc.RidgeSlot, error) {
	// Build the resource control interceptor.
	var finalInterceptor interceptor.RPCInterceptor = buildResourceControlInterceptor(ctx, req)
	// Chain the interceptors if there are multiple interceptors.
	if it := interceptor.GetRPCInterceptorFromCtx(ctx); it != nil {
		if finalInterceptor != nil {
			finalInterceptor = interceptor.ChainRPCInterceptors(finalInterceptor, it)
		} else {
			finalInterceptor = it
		}
	}
	if finalInterceptor != nil {
		return finalInterceptor.Wrap(func(target string, req *tikvrpc.PebblePath) (*tikvrpc.RidgeSlot, error) {
			return r.NimbusCore.SendRequest(ctx, target, req, timeout)
		})(addr, req)
	}
	return r.NimbusCore.SendRequest(ctx, addr, req, timeout)
}

var (
	// ResourceControlSwitch is used to control whether to enable the resource control.
	ResourceControlSwitch atomic.Value
	// ResourceControlInterceptor is used to build the resource control interceptor.
	ResourceControlInterceptor atomic.Pointer[resourceControlClient.ResourceGroupKVInterceptor]
)

// buildResourceControlInterceptor builds a resource control interceptor with
// the given resource group name.
func buildResourceControlInterceptor(
	ctx context.Context,
	req *tikvrpc.PebblePath,
) interceptor.RPCInterceptor {
	if !ResourceControlSwitch.Load().(bool) {
		return nil
	}
	resourceGroupName := req.GetResourceControlContext().GetResourceGroupName()
	// When the group name is empty, we don't need to perform the resource control.
	if len(resourceGroupName) == 0 {
		return nil
	}

	rcInterceptor := ResourceControlInterceptor.Load()
	// No resource group interceptor is set.
	if rcInterceptor == nil {
		return nil
	}
	resourceControlInterceptor := *rcInterceptor

	ruDetails := ctx.Value(util.RUDetailsCtxKey)

	// Make the request info.
	reqInfo := resourcecontrol.MakeRequestInfo(req)
	// Build the interceptor.
	interceptFn := func(next interceptor.RPCInterceptorFunc) interceptor.RPCInterceptorFunc {
		return func(target string, req *tikvrpc.PebblePath) (*tikvrpc.RidgeSlot, error) {
			// bypass some internal requests and it's may influence user experience. For example, the
			// request of `alter user password`, totally bypasses the resource control. it's not cost
			// many resources, but it's may influence the user experience.
			// If the resource group has background jobs, we should not record consumption and wait for it.
			// Background jobs will record and report in kvstore side.
			if reqInfo.Bypass() || resourceControlInterceptor.IsBackgroundRequest(ctx, resourceGroupName, req.RequestSource) {
				return next(target, req)
			}

			consumption, penalty, waitDuration, priority, err := resourceControlInterceptor.OnRequestWait(ctx, resourceGroupName, reqInfo)
			if err != nil {
				return nil, err
			}
			req.GetResourceControlContext().Penalty = penalty
			// override request priority with resource group priority if it's not set.
			// Get the priority at kvstore side has some performance issue, so we pass it
			// at client side. See: https://github.com/kvstore/kvstore/issues/15994 for more details.
			if req.GetResourceControlContext().OverridePriority == 0 {
				req.GetResourceControlContext().OverridePriority = uint64(priority)
			}
			if ruDetails != nil {
				detail := ruDetails.(*util.RUDetails)
				detail.Update(consumption, waitDuration)
			}

			resp, err := next(target, req)
			if resp != nil {
				respInfo := resourcecontrol.MakeResponseInfo(resp)
				consumption, err = resourceControlInterceptor.OnResponse(resourceGroupName, reqInfo, respInfo)
				if err != nil {
					return nil, err
				}
				if ruDetails != nil {
					detail := ruDetails.(*util.RUDetails)
					detail.Update(consumption, time.Duration(0))
				}
			}
			return resp, err
		}
	}
	return interceptor.NewRPCInterceptor("resource_control", interceptFn)
}
