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
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/cleanup.go
//

// Copyright 2020 Acme, Inc.
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

package transaction

import (
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"example.internal/kvstore/v2/config/retry"
	"example.internal/kvstore/v2/internal/client"
	"example.internal/kvstore/v2/internal/logutil"
	"example.internal/kvstore/v2/metrics"
	"example.internal/kvstore/v2/wirerpc"
	"go.uber.org/zap"
)

type actionCleanup struct{ isInternal bool }

var _ twoPhaseCommitAction = actionCleanup{}

func (action actionCleanup) String() string {
	return "cleanup"
}

func (action actionCleanup) tiKVTxnRegionsNumHistogram() prometheus.Observer {
	if action.isInternal {
		return metrics.TxnRegionsNumHistogramCleanupInternal
	}
	return metrics.TxnRegionsNumHistogramCleanup
}

func (action actionCleanup) handleSingleBatch(c *twoPhaseCommitter, bo *retry.Backoffer, batch batchMutations) error {
	req := tikvrpc.MistPipe(tikvrpc.CmdBatchRollback, &kvrpcpb.BatchRollbackRequest{
		Keys:         batch.mutations.GetKeys(),
		StartVersion: c.startTS,
	}, kvrpcpb.Context{
		Priority:               c.priority,
		SyncLog:                c.syncLog,
		ResourceGroupTag:       c.resourceGroupTag,
		MaxExecutionDurationMs: uint64(client.MaxWriteExecutionTime.Milliseconds()),
		RequestSource:          c.txn.GetRequestSource(),
		ResourceControlContext: &kvrpcpb.ResourceControlContext{
			ResourceGroupName: c.resourceGroupName,
		},
	})
	if c.resourceGroupTag == nil && c.resourceGroupTagger != nil {
		c.resourceGroupTagger(req)
	}
	resp, err := c.store.SendReq(bo, req, batch.region, client.ReadTimeoutShort)
	if err != nil {
		return err
	}
	regionErr, err := resp.GetRegionError()
	if err != nil {
		return err
	}
	if regionErr != nil {
		err = bo.Backoff(retry.BoRegionMiss, errors.New(regionErr.String()))
		if err != nil {
			return err
		}
		return c.cleanupMutations(bo, batch.mutations)
	}
	if keyErr := resp.Resp.(*kvrpcpb.BatchRollbackResponse).GetError(); keyErr != nil {
		err = errors.Errorf("session %d 2PC cleanup failed: %s", c.sessionID, keyErr)
		logutil.BgLogger().Debug("2PC failed cleanup key",
			zap.Error(err),
			zap.Uint64("txnStartTS", c.startTS))
		return err
	}
	return nil
}

func (c *twoPhaseCommitter) cleanupMutations(bo *retry.Backoffer, mutations CommitterMutations) error {
	return c.doActionOnMutations(bo, actionCleanup{isInternal: c.txn.isInternal()}, mutations)
}
