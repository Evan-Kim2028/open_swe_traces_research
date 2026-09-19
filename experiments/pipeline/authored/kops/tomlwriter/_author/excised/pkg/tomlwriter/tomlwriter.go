/*
Copyright 2026 The ClusterKit Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package tomlwriter provides a minimal, write-only TOML document builder.
//
// It replaces the frozen github.com/pelletier/go-toml v1 subset used for containerd while matching
// its output byte for byte. Scalars precede subtables, each sorted lexicographically; subtable
// headers have a leading blank line and two-space-per-level indentation. The byte-stable output
// makes the migration off go-toml provably a no-op (the containerdbuilder golden fixtures assert
// exact file contents) and keeps files like /etc/containerd/config.toml identical across upgrades.
package tomlwriter

import (
	_ "fmt"
	_ "maps"
	_ "slices"
	_ "strconv"
	"strings"
)

// Tree represents a TOML document root or table. Its zero value is unusable; use NewTree.
type Tree struct {
	values map[string]any
}

// NewTree returns an empty document.
func NewTree() *Tree {
	panic("excised: NewTree")
}

// Table returns or creates the table at path. To match go-toml v1, a scalar path element leaves
// traversal in the current table.
func (t *Tree) Table(path ...string) *Tree {
	panic("excised: Tree.Table")
}

// SetPath assigns value at path, creating tables with Table's traversal semantics. Path elements
// are unescaped keys, with quoting applied during serialization. Only string, int64, and bool
// values are accepted; other types panic, and a leaf value replaces an existing table.
func (t *Tree) SetPath(path []string, value any) {
	panic("excised: Tree.SetPath")
}

// String serializes the document.
func (t *Tree) String() string {
	panic("excised: Tree.String")
}

// write emits scalars before subtables; keyspace is the dotted, quoted path or empty at root.
func (t *Tree) write(sb *strings.Builder, indent, keyspace string) {
	panic("excised: Tree.write")
}

func scalarString(v any) string {
	panic("excised: scalarString")
}

// quoteKeyIfNeeded leaves A-Za-z0-9_- keys bare and quotes all others as TOML basic strings.
func quoteKeyIfNeeded(k string) string {
	panic("excised: quoteKeyIfNeeded")
}

func isValidBareChar(r rune) bool {
	panic("excised: isValidBareChar")
}

// escapeString matches the go-toml v1 escaping of TOML basic strings.
func escapeString(value string) string {
	panic("excised: escapeString")
}
