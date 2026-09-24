# Closure — packenc

Package: `plumbing/format/packfile`. Files: `encoder.go`, `object_pack.go`.

Removed (29 functions stubbed): `WithObjectSelector`, `NewEncoder`,
`Encoder.Encode`, `Encoder.encode`, `Encoder.head`, `Encoder.entry`,
`Encoder.writeBaseIfDelta`, `Encoder.writeDeltaHeader`,
`Encoder.writeRefDeltaHeader`, `Encoder.writeOfsDeltaHeader`,
`Encoder.entryHead`, `Encoder.footer`, `newOffsetWriter`,
`offsetWriter.Write`, `offsetWriter.Offset`; `newObjectToPack`,
`newDeltaObjectToPack`, `ObjectToPack.BackToOriginal`, `.IsWritten`,
`.MarkWantWrite`, `.WantWrite`, `.SetOriginal`, `.SaveOriginalMetadata`,
`.CleanOriginal`, `.Type`, `.Hash`, `.Size`, `.IsDelta`, `.SetDelta`.

Kept: `ObjectSelector`/`Encoder`/`EncoderOption`/`ObjectToPack`/`offsetWriter`
type declarations and doc comments; the `signature`/`VersionSupported`/
`maskFirstLength`/`firstLengthBits`/`maskContinue`/`maskLength`/`lengthBits`
constants in the untouched siblings; the whole READ side (scanner, parser,
patch-delta applier) stays visible — the wire layout is derivable from the
sibling decoder while the writer's choices are excised.

Tests deleted: all 19 `*_test.go` in the package.
