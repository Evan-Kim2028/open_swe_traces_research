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
	return &parserCache{}
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
	c.oiByHash[oh.Hash] = oh
	c.oiByOffset[oh.Offset] = oh
	c.oi = append(c.oi, oh)
}

func (c *parserCache) Reset(n int) {
	c.oi = make([]*ObjectHeader, 0, n)
	c.oiByHash = make(map[plumbing.Hash]*ObjectHeader, n)
	c.oiByOffset = make(map[int64]*ObjectHeader, n)
}
