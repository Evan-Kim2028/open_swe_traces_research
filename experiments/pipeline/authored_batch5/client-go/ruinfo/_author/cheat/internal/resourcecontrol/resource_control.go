// Copyright 2023 KVStore Authors
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

package resourcecontrol

import (
	"strings"
	"time"

	"example.internal/kvstore/v2/util"
	"example.internal/kvstore/v2/wirerpc"
	_ "github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
)

// RequestInfo contains information about a request that is able to calculate the RU cost
// before the request is sent. Specifically, the write bytes RU cost of a write request
// could be calculated by its key size to write.
type RequestInfo struct {
	// writeBytes is the actual write size if the request is a write request,
	// or -1 if it's a read request.
	writeBytes    int64
	storeID       uint64
	replicaNumber int64
	// bypass indicates whether the request should be bypassed.
	// some internal request should be bypassed, such as Privilege request.
	bypass bool
}

// MakeRequestInfo extracts the relevant information from a BatchRequest.
func MakeRequestInfo(req *tikvrpc.Request) *RequestInfo {
	var bypass bool
	if s := req.Context.GetRequestSource(); strings.Contains(s, util.InternalRequestPrefix+util.InternalTxnOthers) {
		bypass = true
	}
	return &RequestInfo{writeBytes: -1, storeID: req.Context.GetPeer().GetStoreId(), bypass: bypass}
}

// IsWrite returns whether the request is a write request.
func (req *RequestInfo) IsWrite() bool {
	return req.writeBytes > -1
}

// WriteBytes returns the actual write size of the request,
// -1 will be returned if it's not a write request.
func (req *RequestInfo) WriteBytes() uint64 {
	if req.writeBytes > 0 {
		return uint64(req.writeBytes)
	}
	return 0
}

func (req *RequestInfo) ReplicaNumber() int64 {
	return req.replicaNumber
}

// Bypass returns whether the request should be bypassed.
func (req *RequestInfo) Bypass() bool {
	return req.bypass
}

func (req *RequestInfo) StoreID() uint64 {
	return req.storeID
}

// ResponseInfo contains information about a response that is able to calculate the RU cost
// after the response is received. Specifically, the read bytes RU cost of a read request
// could be calculated by its response size, and the KV CPU time RU cost of a request could
// be calculated by its execution details info.
type ResponseInfo struct {
	readBytes uint64
	kvCPU     time.Duration
}

// MakeResponseInfo extracts the relevant information from a BatchResponse.
func MakeResponseInfo(resp *tikvrpc.Response) *ResponseInfo {
	if resp.Resp == nil {
		return &ResponseInfo{}
	}
	var readBytes uint64
	var detailsV2 *kvrpcpb.ExecDetailsV2
	switch r := resp.Resp.(type) {
	case *kvrpcpb.GetResponse:
		detailsV2 = r.GetExecDetailsV2()
	case *kvrpcpb.ScanResponse:
		readBytes = uint64(r.Size())
	default:
		return &ResponseInfo{}
	}
	return &ResponseInfo{readBytes: readBytes, kvCPU: getKVCPU(detailsV2, nil)}
}

// TODO: find out a more accurate way to get the actual KV CPU time.
func getKVCPU(detailsV2 *kvrpcpb.ExecDetailsV2, details *kvrpcpb.ExecDetails) time.Duration {
	if td := detailsV2.GetTimeDetailV2(); td != nil {
		return time.Duration(td.GetProcessWallTimeNs())
	}
	return 0
}

// ReadBytes returns the read bytes of the response.
func (res *ResponseInfo) ReadBytes() uint64 {
	return res.readBytes
}

// KVCPU returns the KV CPU time of the response.
func (res *ResponseInfo) KVCPU() time.Duration {
	return res.kvCPU
}

// Succeed returns whether the KV request is successful.
// Todo: to fit https://github.com/kvstore/meta/pull/5941
func (res *ResponseInfo) Succeed() bool {
	return true
}
