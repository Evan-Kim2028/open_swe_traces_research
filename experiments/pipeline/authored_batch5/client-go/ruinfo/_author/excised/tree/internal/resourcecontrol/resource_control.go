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
	_ "strings"
	"time"

	_ "example.internal/kvstore/v2/util"
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
	panic("excised: MakeRequestInfo")
}

// IsWrite returns whether the request is a write request.
func (req *RequestInfo) IsWrite() bool {
	panic("excised: RequestInfo.IsWrite")
}

// WriteBytes returns the actual write size of the request,
// -1 will be returned if it's not a write request.
func (req *RequestInfo) WriteBytes() uint64 {
	panic("excised: RequestInfo.WriteBytes")
}

func (req *RequestInfo) ReplicaNumber() int64 {
	panic("excised: RequestInfo.ReplicaNumber")
}

// Bypass returns whether the request should be bypassed.
func (req *RequestInfo) Bypass() bool {
	panic("excised: RequestInfo.Bypass")
}

func (req *RequestInfo) StoreID() uint64 {
	panic("excised: RequestInfo.StoreID")
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
	panic("excised: MakeResponseInfo")
}

// TODO: find out a more accurate way to get the actual KV CPU time.
func getKVCPU(detailsV2 *kvrpcpb.ExecDetailsV2, details *kvrpcpb.ExecDetails) time.Duration {
	panic("excised: getKVCPU")
}

// ReadBytes returns the read bytes of the response.
func (res *ResponseInfo) ReadBytes() uint64 {
	panic("excised: ResponseInfo.ReadBytes")
}

// KVCPU returns the KV CPU time of the response.
func (res *ResponseInfo) KVCPU() time.Duration {
	panic("excised: ResponseInfo.KVCPU")
}

// Succeed returns whether the KV request is successful.
// Todo: to fit https://github.com/kvstore/meta/pull/5941
func (res *ResponseInfo) Succeed() bool {
	panic("excised: ResponseInfo.Succeed")
}
