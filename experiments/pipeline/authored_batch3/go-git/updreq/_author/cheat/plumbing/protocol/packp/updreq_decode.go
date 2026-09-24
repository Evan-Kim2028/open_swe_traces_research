package packp

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/format/pktline"
	_ "example.internal/gitkit/v6/plumbing/protocol/capability"
)

var (
	minCommandLength        = sha1HexSize*2 + 2 + 1
	minCommandAndCapsLength = minCommandLength + 1
)

// Decode errors.
var (
	ErrEmpty                        = errors.New("empty update-request message")
	errNoCommands                   = errors.New("unexpected EOF before any command")
	errMissingCapabilitiesDelimiter = errors.New("capabilities delimiter not found")
	errNoFlush                      = errors.New("unexpected EOF before flush line")
)

func errMalformedRequest(reason string) error {
	return fmt.Errorf("malformed request: %s", reason)
}

func errInvalidHash(hash string) error {
	return fmt.Errorf("invalid hash: %s", hash)
}

func errInvalidShallowLineLength(got int) error {
	return errMalformedRequest(fmt.Sprintf(
		"invalid shallow line length: expected %d or %d, got %d",
		len(shallow)+sha1HexSize, len(shallow)+sha256HexSize, got,
	))
}

func errInvalidCommandCapabilitiesLineLength(got int) error {
	return errMalformedRequest(fmt.Sprintf(
		"invalid command and capabilities line length: expected at least %d, got %d",
		minCommandAndCapsLength, got,
	))
}

func errInvalidCommandLineLength(got int) error {
	return errMalformedRequest(fmt.Sprintf(
		"invalid command line length: expected at least %d, got %d",
		minCommandLength, got,
	))
}

func errInvalidShallowObjID(err error) error {
	return errMalformedRequest(
		fmt.Sprintf("invalid shallow object id: %s", err.Error()),
	)
}

func errInvalidOldObjID(err error) error {
	return errMalformedRequest(
		fmt.Sprintf("invalid old object id: %s", err.Error()),
	)
}

func errInvalidNewObjID(err error) error {
	return errMalformedRequest(
		fmt.Sprintf("invalid new object id: %s", err.Error()),
	)
}

func errMalformedCommand(err error) error {
	return errMalformedRequest(fmt.Sprintf(
		"malformed command: %s", err.Error(),
	))
}

// Decode reads the next update-request message from the reader.
//
// https://github.com/git/git/blob/1630431f326e15fcde608827b5ff38422528eb59/builtin/receive-pack.c#L2562-L2566
// https://github.com/git/git/blob/1630431f326e15fcde608827b5ff38422528eb59/pkt-line.c#L466-L493
func (req *UpdateRequests) Decode(r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return ErrEmpty
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		fields := bytes.Fields(line)
		if len(fields) == 3 {
			req.Commands = append(req.Commands, &Command{
				Old:  plumbing.NewHash(string(fields[0])),
				New:  plumbing.NewHash(string(fields[1])),
				Name: plumbing.ReferenceName(fields[2]),
			})
		}
	}
	if len(req.Commands) == 0 {
		return errMalformedRequest("no commands found")
	}
	return nil
}

// parseCommand preserves the complete reference name after the two object IDs.
// See https://github.com/git/git/blob/1630431f326e15fcde608827b5ff38422528eb59/builtin/receive-pack.c#L2144-L2152.
func parseCommand(b []byte) (*Command, error) {
	panic("excised: parseCommand")
}

func parseHash(s string) (plumbing.Hash, error) {
	panic("excised: parseHash")
}
