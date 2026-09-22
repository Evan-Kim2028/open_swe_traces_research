package codegen

import (
	"sort"
	"strings"
	_ "testing"

	_ "github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/http/codegen/testdata"
)

type (
	// httpEndpointExprSnapshot captures the identity of the design
	// attributes that the analyze pass historically rewrote in place.
	// analyze must treat the design expression tree as read-only: the
	// attribute pointers, their types and their marshal tag meta must be
	// unchanged after the services data is computed.
	httpEndpointExprSnapshot struct {
		body              *expr.AttributeExpr
		bodyType          expr.DataType
		streamingBody     *expr.AttributeExpr
		streamingBodyType expr.DataType
		responseBodies    []*expr.AttributeExpr
		responseBodyTypes []expr.DataType
		errorBodies       map[string]*expr.AttributeExpr
		errorBodyTypes    map[string]expr.DataType
	}
)


// collectMarshalTagMeta records the struct:tag:* meta keys carried by the
// attributes reachable from att so the test can assert that analysis does not
// add marshal tags to the design expression tree.
func collectMarshalTagMeta(att *expr.AttributeExpr, tags map[*expr.AttributeExpr][]string, seen map[string]struct{}) {
	if att == nil {
		return
	}
	var keys []string
	for k := range att.Meta {
		if strings.HasPrefix(k, "struct:tag:") {
			keys = append(keys, k)
		}
	}
	if len(keys) > 0 {
		sort.Strings(keys)
		tags[att] = keys
	}
	switch dt := att.Type.(type) {
	case expr.UserType:
		if _, ok := seen[dt.ID()]; ok {
			return
		}
		seen[dt.ID()] = struct{}{}
		collectMarshalTagMeta(dt.Attribute(), tags, seen)
	case *expr.Object:
		for _, nat := range *dt {
			collectMarshalTagMeta(nat.Attribute, tags, seen)
		}
	case *expr.Array:
		collectMarshalTagMeta(dt.ElemType, tags, seen)
	case *expr.Map:
		collectMarshalTagMeta(dt.KeyType, tags, seen)
		collectMarshalTagMeta(dt.ElemType, tags, seen)
	case *expr.Union:
		for _, nat := range dt.Values {
			collectMarshalTagMeta(nat.Attribute, tags, seen)
		}
	}
}
