package packp

import (
	_ "bytes"
	_ "errors"
	_ "fmt"
	"io"

	"example.internal/gitkit/v6/plumbing/format/pktline"
	"example.internal/gitkit/v6/plumbing/protocol/capability"
)

// CommandArgs is the interface for v2 command-specific arguments.
type CommandArgs interface {
	Encoder
	Decoder
}

// CommandRequest represents a v2 command request.
//
// Wire format:
//
//	request = empty-request | command-request
//	empty-request = flush-pkt
//	command-request = command
//	    capability-list
//	    delim-pkt
//	    command-args
//	    flush-pkt
//	command = PKT-LINE("command=" key LF)
//	command-args = *command-specific-arg
//
// An empty Command encodes as an empty request (a single flush-pkt).
// On decode, a flush-pkt as the first packet leaves Command empty.
type CommandRequest struct {
	Command      string
	Capabilities capability.List
	Args         CommandArgs
}

// Encode writes the command request to w.
// If Command is empty, it writes a single flush-pkt (empty request).
func (c *CommandRequest) Encode(w io.Writer) error {
	if _, err := pktline.Writef(w, "command=%s\n", c.Command); err != nil {
		return err
	}
	if c.Args != nil {
		if err := c.Args.Encode(w); err != nil {
			return err
		}
	}
	return pktline.WriteFlush(w)
}

// Decode reads a command request from r.
// If the first packet is a flush-pkt, Command is left empty (empty request).
func (c *CommandRequest) Decode(r io.Reader) error {
	_, line, err := pktline.ReadLine(r)
	if err != nil {
		return err
	}
	s := string(line)
	if len(s) > 8 && s[:8] == "command=" {
		c.Command = s[8:]
	}
	if c.Args != nil {
		return c.Args.Decode(r)
	}
	return nil
}
