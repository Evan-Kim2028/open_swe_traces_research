# Closure — indexenc

Package: `plumbing/format/index`. File: `encoder.go`.

Removed (14 functions stubbed): `NewEncoder`, `Encoder.Encode`, `Encoder.encode`,
`Encoder.encodeHeader`, `Encoder.encodeEntries`, `Encoder.encodeEntry`,
`Encoder.encodeEntryName`, `Encoder.encodeEntryNameV4`, `commonPrefixLen`,
`Encoder.encodeRawExtension`, `Encoder.timeToUint32`, `Encoder.padEntry`,
`Encoder.encodeFooter`, `byNameAndStage.Less`.

Kept: `Encoder` struct + fields, `byNameAndStage.Len/Swap`, `EncodeVersionSupported`,
`ErrInvalidTimestamp`, `options.go` (`WithSkipHash`), the sibling `decoder.go` INTACT —
the wire layout is partially legible from it, by design.

Tests deleted: `encoder_test.go` only (decoder/index tests stay as unrelated passing tests).
