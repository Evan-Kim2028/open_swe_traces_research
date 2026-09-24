// This file verifies that adding OpenAPI extensions does not change values in
// the object being encoded.
package openapi

import (
	_ "bytes"
	_ "encoding/json"
	_ "testing"

	_ "github.com/stretchr/testify/require"
)

type (
	marshalExample struct {
		Value int64 `json:"value"`
	}
)

