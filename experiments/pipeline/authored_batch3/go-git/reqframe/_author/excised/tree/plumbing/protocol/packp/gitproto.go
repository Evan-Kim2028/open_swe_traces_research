package packp

import (
	"fmt"
	"io"
	_ "strings"

	_ "example.internal/gitkit/v6/plumbing/format/pktline"
)

// ErrInvalidGitProtoRequest is returned by Decode if the input is not a
// valid git protocol request.
var ErrInvalidGitProtoRequest = fmt.Errorf("invalid git protocol request")

// GitProtoRequest is a command request for the git protocol.
// It is used to send the command, endpoint, and extra parameters to the
// remote.
// See https://git-scm.com/docs/pack-protocol#_git_transport
type GitProtoRequest struct {
	RequestCommand string
	Pathname       string

	// Optional
	Host string

	// Optional
	ExtraParams []string
}

// validate validates the request.
func (g *GitProtoRequest) validate() error {
	panic("excised: GitProtoRequest.validate")
}

// validateGitProtoField rejects a request field containing an ASCII control
// byte (0x00-0x1f or 0x7f). Such a byte would break the NUL-framed pkt-line or
// splice in additional fields. No valid command, path, host, or parameter
// contains one. This matches upstream git, which forbids newlines in the host
// and path of a git:// request (git.git a02ea577, CVE-2021-40330), and extends
// it to the full control range including NUL, which a Go string can carry
// through where a C string cannot.
func validateGitProtoField(name, value string) error {
	panic("excised: validateGitProtoField")
}

// Encode encodes the request into the writer.
func (g *GitProtoRequest) Encode(w io.Writer) error {
	panic("excised: GitProtoRequest.Encode")
}

// Decode decodes the request from the reader.
func (g *GitProtoRequest) Decode(r io.Reader) error {
	panic("excised: GitProtoRequest.Decode")
}
