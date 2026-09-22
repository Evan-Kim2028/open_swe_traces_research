# Exported API — packlookup

Package `plumbing/format/packfile` — the random-access read path:
`NewPackfile`/`WithPackHandle`, `Packfile.Get`/`GetByOffset`/
`GetSizeByOffset`/`GetAll`/`GetByType`/`Scanner`/`ID`/`Close`,
`init`/`get`/`getByOffset`/`headerFromOffset`/`objectFromHeader`/
`getMemoryObject`/`openRandomReader`, `FSObject` (`NewFSObject`,
`Reader`, accessors) and `objectIter` (`Next`/`ForEach`/`Close`).

Kept visible: `Packfile`/`FSObject`/`objectIter` types, `PackHandle`/
`RandomReader`/`PackHandleResolver` interfaces and their contract
comments, `ErrInvalidObject`/`ErrZLib`, `probePack` docs; the scanner
(`packscan`), parser (`packparse`), index (`idxindex`) and delta codecs
(`packdelta`, `deltadiff`) are intact.

Callers: filesystem storage, transport pack consumers. In-tree tests
removed: 3 (`packfile_test.go`, `fsobject_test.go`, `internal_test.go`).
