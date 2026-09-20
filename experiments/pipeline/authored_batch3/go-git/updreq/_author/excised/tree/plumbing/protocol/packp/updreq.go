package packp

import (
	"errors"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/protocol/capability"
)

// Errors returned by the updreq package.
var (
	ErrEmptyCommands    = errors.New("commands cannot be empty")
	ErrMalformedCommand = errors.New("malformed command")
)

// UpdateRequests values represent reference upload requests.
// The zero value is safe to use; Commands and Shallows can be populated
// via append.
type UpdateRequests struct {
	Capabilities capability.List
	Commands     []*Command
	Shallows     []plumbing.Hash
	// TODO: Support push-cert
}

func validateUpdateRequests(req *UpdateRequests) error {
	panic("excised: validateUpdateRequests")
}

// Action represents the action type of a command.
type Action string

// Action types.
const (
	Create  Action = "create"
	Update  Action = "update"
	Delete  Action = "delete"
	Invalid Action = "invalid"
)

// Command represents a command to be executed on a reference.
type Command struct {
	Name plumbing.ReferenceName
	Old  plumbing.Hash
	New  plumbing.Hash
}

// Action returns the action type of the command.
func (c *Command) Action() Action {
	panic("excised: Command.Action")
}

func (c *Command) validate() error {
	panic("excised: Command.validate")
}
