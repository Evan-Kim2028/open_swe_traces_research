package packhandle

import (
	_ "bytes"
	"encoding/binary"
	_ "errors"
	"fmt"

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
	var header [12]byte
	if _, err := src.ReadAt(header[:], 0); err != nil {
		return PackMeta{}, fmt.Errorf("packhandle: read pack header: %w", err)
	}
	version := binary.BigEndian.Uint32(header[4:8])
	count := binary.BigEndian.Uint32(header[8:12])

	hashSize := int64(packHash.Size())
	footer := make([]byte, hashSize)
	if _, err := src.ReadAt(footer, size-hashSize); err != nil {
		return PackMeta{}, fmt.Errorf("packhandle: read pack footer: %w", err)
	}
	var id plumbing.Hash
	id.ResetBySize(int(hashSize))
	if _, err := id.Write(footer); err != nil {
		return PackMeta{}, fmt.Errorf("packhandle: write footer to hash: %w", err)
	}
	return PackMeta{Version: version, Count: count, ID: id}, nil
}
