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
// https://github.com/acme/sqlengine/tree/cc5e161ac06827589c4966674597c137cc9e809c/store/kvstore/logutil/hex.go
//

// Copyright 2021 Acme, Inc.
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

package logutil

import (
	_ "bytes"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	_ "strings"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
)

// Hex defines a fmt.Stringer for proto.Message.
// We can't define the String() method on proto.Message, but we can wrap it.
func Hex(msg proto.Message) fmt.Stringer {
	return hexStringer{msg}
}

type hexStringer struct {
	proto.Message
}

func (h hexStringer) String() string {
	switch m := h.Message.(type) {
	case *metapb.Peer:
		if m == nil {
			return "<nil>"
		}
		return fmt.Sprintf("{Id:%d StoreId:%d Role:%v IsWitness:%v}", m.Id, m.StoreId, m.Role, m.IsWitness)
	case *kvrpcpb.GetRequest:
		return fmt.Sprintf("{Context:%v Key:%s Version:%d}", m.Context, hex.EncodeToString(m.Key), m.Version)
	default:
		return fmt.Sprintf("%v", h.Message)
	}
}

func prettyPrint(w io.Writer, val reflect.Value) {
	panic("excised: prettyPrint")
}
