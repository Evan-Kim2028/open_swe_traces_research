package packp

import (
	_ "fmt"
	"io"

	_ "example.internal/gitkit/v6/plumbing/format/pktline"
	"example.internal/gitkit/v6/plumbing/protocol/capability"
)

// Encode writes the ReferenceUpdateRequest encoding to the stream.
func (req *UpdateRequests) Encode(w io.Writer) error {
	panic("excised: UpdateRequests.Encode")
}

func (req *UpdateRequests) encodeShallow(w io.Writer) error {
	panic("excised: UpdateRequests.encodeShallow")
}

func (req *UpdateRequests) encodeCommands(w io.Writer,
	cmds []*Command, caps *capability.List,
) error {
	panic("excised: UpdateRequests.encodeCommands")
}

func formatCommand(cmd *Command) string {
	panic("excised: formatCommand")
}
