package packhandle

import (
	_ "bytes"
	_ "encoding/binary"
	_ "errors"
	_ "fmt"

	"example.internal/gitkit/v6/plumbing"
)

// PackMeta is the parsed pack header plus footer hash.
type PackMeta struct {
	Version uint32        // pack format version, validated to be 2 or 3
	Count   uint32        // number of objects in the pack
	ID      plumbing.Hash // pack footer hash
}

var packMagic = []byte{'P', 'A', 'C', 'K'}

// parsePackMeta reads and validates the 12-byte pack header and
// the footer hash at the tail. The returned [PackMeta] is
// well-formed only if the footer equals packHash.
func parsePackMeta(src ReadAtCloser, size int64, packHash plumbing.Hash) (PackMeta, error) {
	panic("excised: parsePackMeta")
}
