package packp

import (
	"errors"
	_ "fmt"
	"io"
	_ "strings"
	_ "unicode"

	_ "example.internal/gitkit/v6/plumbing/format/pktline"
)

// ErrInvalidPushOption is returned when a push option contains invalid
// characters.
var ErrInvalidPushOption = errors.New("invalid push option")

// PushOptions represents a list of update request push-options.
//
// See https://git-scm.com/docs/gitprotocol-pack#_reference_update_request_and_packfile_transfer
type PushOptions struct {
	Options []string
}

// Encode encodes the push options into the given writer.
func (opts *PushOptions) Encode(w io.Writer) error {
	panic("excised: PushOptions.Encode")
}

// Decode decodes the push options from the given reader.
func (opts *PushOptions) Decode(r io.Reader) error {
	panic("excised: PushOptions.Decode")
}

func isNotGraphic(r rune) bool {
	panic("excised: isNotGraphic")
}
