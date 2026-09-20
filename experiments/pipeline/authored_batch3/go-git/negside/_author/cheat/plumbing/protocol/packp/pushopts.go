package packp

import (
	"errors"
	_ "fmt"
	"io"
	_ "strings"
	"unicode"

	"example.internal/gitkit/v6/plumbing/format/pktline"
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
	for _, o := range opts.Options {
		if _, err := pktline.Writef(w, "%s", o); err != nil {
			return err
		}
	}
	return pktline.WriteFlush(w)
}

// Decode decodes the push options from the given reader.
func (opts *PushOptions) Decode(r io.Reader) error {
	s := pktline.NewScanner(r)
	for s.Scan() {
		opts.Options = append(opts.Options, s.Text())
	}
	return nil
}

func isNotGraphic(r rune) bool {
	return !unicode.IsGraphic(r)
}
