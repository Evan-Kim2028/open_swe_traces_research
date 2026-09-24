# Closure — packlookup

Package: `plumbing/format/packfile`. Files: `packfile.go`,
`fsobject.go`, `packfile_iter.go`, `pack_handle.go`.

Removed (32 functions stubbed): `NewPackfile`, `Packfile.Get`,
`.GetByOffset`, `.GetSizeByOffset`, `.GetAll`, `.GetByType`, `.Scanner`,
`.ID`, `.get`, `.getByOffset`, `.init`, `.headerFromOffset`, `.Close`,
`.objectFromHeader`, `.getMemoryObject`, `.openRandomReader`;
`WithPackHandle`; `probePack`, `NewFSObject`, `FSObject.Reader`,
`.SetSize`, `.SetType`, `.Hash`, `.Size`, `.Type`, `.Writer`,
`zlibReadCloser.Read`, `.Close`; `objectIter.Next`, `.next`, `.ForEach`,
`.Close`.

Kept: `Packfile`/`FSObject`/`objectIter`/`PackHandle`/`RandomReader`/
`PackHandleResolver` types and their contract comments, `probeSize`/
`probeBufPool`, `ErrInvalidObject`/`ErrZLib`, `common.go` helpers,
`types.go`. Disjoint from packscan (wire scanner), packparse (parser
state machine), packenc (encoder), idxindex (the index structure),
packhandle (the internal/packhandle FD pool) and packdelta/deltadiff
(delta codecs) — this is the read dispatch that sits over all of them.

Tests deleted: `packfile_test.go`, `fsobject_test.go`,
`internal_test.go` (3 — the tests exercising this read path; encoder,
parser, scanner, delta and selector suites stay).
