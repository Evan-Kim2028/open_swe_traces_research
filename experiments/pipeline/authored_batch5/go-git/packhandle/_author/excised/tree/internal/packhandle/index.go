package packhandle

import (
	_ "io/fs"

	"example.internal/gitkit/v6/plumbing/format/idxfile"
)

// Index returns a lazily-constructed [idxfile.Index] backed by
// the Idx and Rev sources. The first successful build is cached;
// transient build failures are not cached and retry on the next
// call. Returns [ErrSourceUnconfigured] if Idx or Rev was not
// configured, or [fs.ErrClosed] if the [PackHandle] is closed.
func (h *PackHandle) Index() (idxfile.Index, error) {
	panic("excised: PackHandle.Index")
}
