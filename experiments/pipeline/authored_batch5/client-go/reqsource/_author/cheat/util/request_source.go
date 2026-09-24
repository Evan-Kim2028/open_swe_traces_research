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

package util

import (
	"context"
	"strings"
)

// RequestSourceTypeKeyType is a dummy type to avoid naming collision in context.
type RequestSourceTypeKeyType struct{}

// RequestSourceTypeKey is used as the key of request source type in context.
var RequestSourceTypeKey = RequestSourceTypeKeyType{}

// RequestSourceKeyType is a dummy type to avoid naming collision in context.
type RequestSourceKeyType struct{}

// RequestSourceKey is used as the key of request source type in context.
var RequestSourceKey = RequestSourceKeyType{}

const (
	// InternalTxnOthers is the type of requests that consume low resources.
	// This reduces the size of metrics.
	InternalTxnOthers = "others"
	// InternalTxnGC is the type of GC txn.
	InternalTxnGC = "gc"
	// InternalTxnMeta is the type of the miscellaneous meta usage.
	InternalTxnMeta = InternalTxnOthers
)

// explicit source types.
const (
	ExplicitTypeEmpty      = ""
	ExplicitTypeLightning  = "lightning"
	ExplicitTypeBR         = "br"
	ExplicitTypeDumpling   = "dumpling"
	ExplicitTypeBackground = "background"
	ExplicitTypeDDL        = "ddl"
	ExplicitTypeStats      = "stats"
)

// ExplicitTypeList is the list of all explicit source types.
var ExplicitTypeList = []string{ExplicitTypeEmpty, ExplicitTypeLightning, ExplicitTypeBR, ExplicitTypeDumpling, ExplicitTypeBackground, ExplicitTypeDDL, ExplicitTypeStats}

const (
	// InternalRequest is the scope of internal queries
	InternalRequest = "internal"
	// InternalRequestPrefix is the prefix of internal queries
	InternalRequestPrefix = "internal_"
	// ExternalRequest is the scope of external queries
	ExternalRequest = "external"
	// SourceUnknown keeps same with the default value(empty string)
	SourceUnknown = "unknown"
)

// RequestSource contains the source label of the request, used for tracking resource consuming.
type RequestSource struct {
	RequestSourceInternal bool
	RequestSourceType     string
	// ExplicitRequestSourceType is a type that is set from the session variable and may be specified by the client or users.
	// It is a complement to the RequestSourceType and provides additional information about how a request was initiated.
	ExplicitRequestSourceType string
}

// SetRequestSourceInternal sets the scope of the request source.
func (r *RequestSource) SetRequestSourceInternal(internal bool) {
	panic("excised: RequestSource.SetRequestSourceInternal")
}

// SetRequestSourceType sets the type of the request source.
func (r *RequestSource) SetRequestSourceType(tp string) {
	panic("excised: RequestSource.SetRequestSourceType")
}

// SetExplicitRequestSourceType sets the type of the request source.
func (r *RequestSource) SetExplicitRequestSourceType(tp string) {
	panic("excised: RequestSource.SetExplicitRequestSourceType")
}

// WithInternalSourceType create context with internal source.
func WithInternalSourceType(ctx context.Context, source string) context.Context {
	return context.WithValue(ctx, RequestSourceKey, RequestSource{RequestSourceInternal: true, RequestSourceType: source})
}

// WithInternalSourceAndTaskType create context with internal source and task name.
func WithInternalSourceAndTaskType(ctx context.Context, source, taskName string) context.Context {
	panic("excised: WithInternalSourceAndTaskType")
}

// BuildRequestSource builds a request_source from internal, source and explicitSource.
func BuildRequestSource(internal bool, source, explicitSource string) string {
	return (&RequestSource{internal, source, explicitSource}).GetRequestSource()
}

// IsRequestSourceInternal checks whether the input request source type is internal type.
func IsRequestSourceInternal(reqSrc *RequestSource) bool {
	return reqSrc != nil && IsInternalRequest(reqSrc.GetRequestSource())
}

// GetRequestSource gets the request_source field of the request.
func (r *RequestSource) GetRequestSource() string {
	if r == nil || (r.RequestSourceType == "" && r.ExplicitRequestSourceType == "") {
		return SourceUnknown
	}
	origin := ExternalRequest
	if r.RequestSourceInternal {
		origin = InternalRequest
	}
	src := r.RequestSourceType
	if src == "" {
		src = SourceUnknown
	}
	out := origin + "_" + src
	if r.ExplicitRequestSourceType != "" && r.ExplicitRequestSourceType != r.RequestSourceType {
		out += "_" + r.ExplicitRequestSourceType
	}
	return out
}

// RequestSourceFromCtx extract source from passed context.
func RequestSourceFromCtx(ctx context.Context) string {
	panic("excised: RequestSourceFromCtx")
}

// IsInternalRequest returns the type of the request source.
func IsInternalRequest(source string) bool {
	return strings.HasPrefix(source, "internal")
}

// ResourceGroupNameKeyType is the context key type of resource group name.
type resourceGroupNameKeyType struct{}

// ResourceGroupNameKey is used as the key of request source type in context.
var resourceGroupNameKey = resourceGroupNameKeyType{}

// WithResourceGroupName return a copy of the given context with a associated
// resource group name.
func WithResourceGroupName(ctx context.Context, groupName string) context.Context {
	panic("excised: WithResourceGroupName")
}

// ResourceGroupNameFromCtx extract resource group name from passed context,
// empty string is returned is the key is not set.
func ResourceGroupNameFromCtx(ctx context.Context) string {
	panic("excised: ResourceGroupNameFromCtx")
}
