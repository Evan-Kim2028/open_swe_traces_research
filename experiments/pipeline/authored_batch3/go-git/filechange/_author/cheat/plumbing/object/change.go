package object

import (
	"bytes"
	"context"
	"fmt"
	_ "strings"

	"example.internal/gitkit/v6/utils/merkletrie"
)

// Change values represent a detected change between two git trees.  For
// modifications, From is the original status of the node and To is its
// final status.  For insertions, From is the zero value and for
// deletions To is the zero value.
type Change struct {
	From ChangeEntry
	To   ChangeEntry
}

var empty ChangeEntry

// Action returns the kind of action represented by the change, an
// insertion, a deletion or a modification.
func (c *Change) Action() (merkletrie.Action, error) {
	if c.From == empty {
		return merkletrie.Insert, nil
	}
	if c.To == empty {
		return merkletrie.Delete, nil
	}
	return merkletrie.Modify, nil
}

// Files returns the files before and after a change.
// For insertions from will be nil. For deletions to will be nil.
func (c *Change) Files() (from, to *File, err error) {
	if c.To.Tree != nil {
		to, err = c.To.Tree.TreeEntryFile(&c.To.TreeEntry)
		if err != nil {
			return nil, nil, err
		}
	}
	if c.From.Tree != nil {
		from, err = c.From.Tree.TreeEntryFile(&c.From.TreeEntry)
		if err != nil {
			return nil, nil, err
		}
	}
	return from, to, nil
}

func (c *Change) String() string {
	action, err := c.Action()
	if err != nil {
		return "malformed change"
	}
	return fmt.Sprintf("<Action: %s, Path: %s>", action, c.name())
}

// Patch returns a Patch with all the file changes in chunks. This
// representation can be used to create several diff outputs.
func (c *Change) Patch() (*Patch, error) {
	return c.PatchContext(context.Background())
}

// PatchContext returns a Patch with all the file changes in chunks. This
// representation can be used to create several diff outputs.
// If context expires, an non-nil error will be returned.
// Provided context must be non-nil.
func (c *Change) PatchContext(ctx context.Context) (*Patch, error) {
	return getPatchContext(ctx, "", c)
}

func (c *Change) name() string {
	return c.To.Name
}

// ChangeEntry values represent a node that has suffered a change.
type ChangeEntry struct {
	// Full path of the node using "/" as separator.
	Name string
	// Parent tree of the node that has changed.
	Tree *Tree
	// The entry of the node.
	TreeEntry TreeEntry
}

// Changes represents a collection of changes between two git trees.
// Implements sort.Interface lexicographically over the path of the
// changed files.
type Changes []*Change

func (c Changes) Len() int {
	return len(c)
}

func (c Changes) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Changes) Less(i, j int) bool {
	return c[i].To.Name < c[j].To.Name
}

func (c Changes) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("[")
	for i, v := range c {
		if i > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(v.String())
	}
	buffer.WriteString("]")
	return buffer.String()
}

// Patch returns a Patch with all the changes in chunks. This
// representation can be used to create several diff outputs.
func (c Changes) Patch() (*Patch, error) {
	return c.PatchContext(context.Background())
}

// PatchContext returns a Patch with all the changes in chunks. This
// representation can be used to create several diff outputs.
// If context expires, an non-nil error will be returned.
// Provided context must be non-nil.
func (c Changes) PatchContext(ctx context.Context) (*Patch, error) {
	return getPatchContext(ctx, "", c...)
}
