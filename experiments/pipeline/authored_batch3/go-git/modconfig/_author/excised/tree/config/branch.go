package config

import (
	"errors"
	_ "strings"

	"example.internal/gitkit/v6/plumbing"
	format "example.internal/gitkit/v6/plumbing/format/config"
)

var (
	errBranchEmptyName     = errors.New("branch config: empty name")
	errBranchInvalidRebase = errors.New("branch config: rebase must be one of 'true' or 'interactive'")
)

// Branch contains information on the
// local branches and which remote to track
type Branch struct {
	// Name of branch
	Name string
	// Remote name of remote to track
	Remote string
	// Merge is the local refspec for the branch
	Merge plumbing.ReferenceName
	// Rebase instead of merge when pulling. Valid values are
	// "true" and "interactive".  "false" is undocumented and
	// typically represented by the non-existence of this field
	Rebase string
	// Description explains what the branch is for.
	// Multi-line explanations may be used.
	//
	// Original git command to edit:
	//	git branch --edit-description
	Description string

	raw *format.Subsection
}

// Validate validates fields of branch.
// It does not reject Merge values that lack a "refs/" prefix,
// matching the behaviour of real git during config read and write.
func (b *Branch) Validate() error {
	panic("excised: Branch.Validate")
}

func (b *Branch) marshal() *format.Subsection {
	panic("excised: Branch.marshal")
}

// hack to trigger conditional quoting in the
// plumbing/format/config/Encoder.encodeOptions
//
// Current Encoder implementation uses Go %q format if value contains a backslash character,
// which is not consistent with reference git implementation.
// git just replaces newline characters with \n, while Encoder prints them directly.
// Until value quoting fix, we should escape description value by replacing newline characters with \n.
func quoteDescription(desc string) string {
	panic("excised: quoteDescription")
}

func (b *Branch) unmarshal(s *format.Subsection) error {
	panic("excised: Branch.unmarshal")
}

// hack to enable conditional quoting in the
// plumbing/format/config/Encoder.encodeOptions
// goto quoteDescription for details.
func unquoteDescription(desc string) string {
	panic("excised: unquoteDescription")
}
