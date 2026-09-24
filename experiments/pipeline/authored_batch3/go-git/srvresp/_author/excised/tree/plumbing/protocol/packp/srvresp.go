package packp

import (
	_ "bytes"
	_ "errors"
	_ "fmt"
	"io"
	_ "strings"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/format/pktline"
)

const ackLineLen = 44

// ServerResponse object acknowledgement from upload-pack service
type ServerResponse struct {
	ACKs []ACK
}

// ACKStatus represents the status of an object acknowledgement.
type ACKStatus byte

// String returns the string representation of the ACKStatus.
func (s ACKStatus) String() string {
	panic("excised: ACKStatus.String")
}

// ACKStatus values
const (
	ACKContinue ACKStatus = iota + 1
	ACKCommon
	ACKReady
)

// ACK represents an object acknowledgement. A status can be zero when the
// response doesn't support multi_ack and multi_ack_detailed capabilities.
type ACK struct {
	Hash   plumbing.Hash
	Status ACKStatus
}

// Decode decodes the response into the struct.
func (r *ServerResponse) Decode(reader io.Reader) error {
	panic("excised: ServerResponse.Decode")
}

func (r *ServerResponse) decodeLine(line []byte) error {
	panic("excised: ServerResponse.decodeLine")
}

func (r *ServerResponse) decodeACKLine(line []byte) (err error) {
	panic("excised: ServerResponse.decodeACKLine")
}

// Encode encodes the ServerResponse into a writer.
func (r *ServerResponse) Encode(w io.Writer) error {
	panic("excised: ServerResponse.Encode")
}

// encodeServerResponse encodes the ServerResponse into a writer.
func encodeServerResponse(w io.Writer, acks []ACK) error {
	panic("excised: encodeServerResponse")
}
