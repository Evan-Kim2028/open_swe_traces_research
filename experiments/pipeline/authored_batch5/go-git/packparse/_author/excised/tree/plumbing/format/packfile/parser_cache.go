package packfile

import (
	_ "slices"

	"example.internal/gitkit/v6/plumbing"
)

// maxObjectsPrealloc caps the up-front capacity reserved from the pack's
// declared object count, so a header advertising an absurd quantity cannot
// trigger a multi-gigabyte allocation. The slice and maps still grow
// organically beyond this hint.
const maxObjectsPrealloc = 1 << 16 // 64 Ki entries

func newParserCache() *parserCache {
	panic("excised: newParserCache")
}

// parserCache defines the cache used within the parser.
// This is not thread safe by itself, and relies on the parser to
// enforce thread-safety.
type parserCache struct {
	oi         []*ObjectHeader
	oiByHash   map[plumbing.Hash]*ObjectHeader
	oiByOffset map[int64]*ObjectHeader
}

func (c *parserCache) Add(oh *ObjectHeader) {
	panic("excised: parserCache.Add")
}

func (c *parserCache) Reset(n int) {
	panic("excised: parserCache.Reset")
}
