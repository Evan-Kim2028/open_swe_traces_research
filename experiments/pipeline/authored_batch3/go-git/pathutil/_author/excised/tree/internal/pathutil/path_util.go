// Package pathutil provides path utility functions.
package pathutil

import (
	_ "os"
	_ "os/user"
	_ "strings"
)

// ReplaceTildeWithHome replaces the tilde character at the beginning of a path
// with the appropriate home directory.
func ReplaceTildeWithHome(path string) (string, error) {
	panic("excised: ReplaceTildeWithHome")
}

// HasUnsafeComponent reports whether name, split on '/' and '\\', has a path
// component that must not be stored verbatim as a filesystem path: one holding
// a control character, or one that a case-insensitive, NTFS or HFS+ filesystem
// would resolve back to "." or "..".
//
// A literal ".." is only the plainest spelling of an escape. IsHFSDot and
// IsNTFSDot read their needle as the component's spelling after the leading
// dot, so "." finds ".." and "" finds a bare "." — each with the disguises
// those two already cover: the code points HFS+ drops during normalisation,
// and the trailing dots or spaces NTFS trims and the Alternate Data Stream
// suffix it truncates at. A component folding to ".." aliases the parent
// directory, and one folding to "." aliases the directory it sits in, which
// turns "refs/heads/<U+200C>./main" into "refs/heads/main".
//
// Three of the four needle combinations do work. IsNTFSDot with an empty
// needle already matches every component NTFS trims down to a dot, and what it
// matches strictly contains what the "." needle could add, so that fourth
// call would never be the one to fire.
//
// A component holding no '.' can match none of them, so the control-character
// scan records whether it saw a dot and the fold checks run only if it did.
// The checks run regardless of the host OS, because a name can be authored on
// one system and reach the filesystem on another.
func HasUnsafeComponent(name string) bool {
	panic("excised: HasUnsafeComponent")
}

func isPathSep(r rune) bool {
	panic("excised: isPathSep")
}
