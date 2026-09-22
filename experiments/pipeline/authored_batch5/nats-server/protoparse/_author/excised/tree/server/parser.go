// Copyright 2012-2026 The Msgkit Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	_ "bufio"
	_ "bytes"
	_ "fmt"
	"net/http"
	_ "net/textproto"
)

type parserState int
type parseState struct {
	state   parserState
	op      byte
	as      int
	drop    int
	pa      pubArg
	argBuf  []byte
	msgBuf  []byte
	header  http.Header // access via getHeader
	scratch [MAX_CONTROL_LINE_SIZE]byte
	argsa   [MAX_HMSG_ARGS + 1][]byte // pre-allocated args array to avoid per-call heap escape
}

type pubArg struct {
	arg       []byte
	pacache   []byte
	origin    []byte
	account   []byte
	subject   []byte
	deliver   []byte
	mapped    []byte
	reply     []byte
	szb       []byte
	hdb       []byte
	queues    [][]byte
	size      int
	hdr       int
	psi       []*serviceImport
	trace     *msgTrace
	delivered bool // Only used for service imports
}

// Parser constants
const (
	OP_START parserState = iota
	OP_PLUS
	OP_PLUS_O
	OP_PLUS_OK
	OP_MINUS
	OP_MINUS_E
	OP_MINUS_ER
	OP_MINUS_ERR
	OP_MINUS_ERR_SPC
	MINUS_ERR_ARG
	OP_C
	OP_CO
	OP_CON
	OP_CONN
	OP_CONNE
	OP_CONNEC
	OP_CONNECT
	CONNECT_ARG
	OP_H
	OP_HP
	OP_HPU
	OP_HPUB
	OP_HPUB_SPC
	HPUB_ARG
	OP_HM
	OP_HMS
	OP_HMSG
	OP_HMSG_SPC
	HMSG_ARG
	OP_P
	OP_PU
	OP_PUB
	OP_PUB_SPC
	PUB_ARG
	OP_PI
	OP_PIN
	OP_PING
	OP_PO
	OP_PON
	OP_PONG
	MSG_PAYLOAD
	MSG_END_R
	MSG_END_N
	OP_S
	OP_SU
	OP_SUB
	OP_SUB_SPC
	SUB_ARG
	OP_A
	OP_ASUB
	OP_ASUB_SPC
	ASUB_ARG
	OP_AUSUB
	OP_AUSUB_SPC
	AUSUB_ARG
	OP_L
	OP_LS
	OP_R
	OP_RS
	OP_U
	OP_UN
	OP_UNS
	OP_UNSU
	OP_UNSUB
	OP_UNSUB_SPC
	UNSUB_ARG
	OP_M
	OP_MS
	OP_MSG
	OP_MSG_SPC
	MSG_ARG
	OP_I
	OP_IN
	OP_INF
	OP_INFO
	INFO_ARG
)

func (c *client) parse(buf []byte) error { panic("excised: parse") }

func protoSnippet(start, max int, buf []byte) string { panic("excised: protoSnippet") }

// Check if the length of buffer `arg` is over the max control line limit `mcl`.
// If so, an error is sent to the client and the connection is closed.
// The error ErrMaxControlLine is returned.
func (c *client) overMaxControlLineLimit(arg []byte, mcl int32) error { panic("excised: overMaxControlLineLimit") }

// clonePubArg is used when the split buffer scenario has the pubArg in the existing read buffer, but
// we need to hold onto it into the next read.
func (c *client) clonePubArg(lmsg bool) error { panic("excised: clonePubArg") }

func (ps *parseState) getHeader() http.Header { panic("excised: getHeader") }
