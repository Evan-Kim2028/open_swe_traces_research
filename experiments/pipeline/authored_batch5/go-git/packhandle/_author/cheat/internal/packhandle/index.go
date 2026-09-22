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
	idxOpener := func() (idxfile.ReadAtCloser, error) { return h.sources.Idx.Open() }
	revOpener := func() (idxfile.ReadAtCloser, error) { return h.sources.Rev.Open() }
	return idxfile.NewLazyIndex(idxOpener, revOpener, h.packHash)
}
