# Closure — capability

Package: `plumbing/protocol/capability`. Files: `list.go`, `capability.go`.

Removed (19 functions stubbed): `DecodeList`, `EncodeList`, `List.Set`, `List.init`,
`List.Add`, `List.Delete`, `List.All`, `List.MarshalText`, `List.AppendText`,
`List.UnmarshalText`, `List.String`, `DefaultAgent`, `Validate`, `validateCapability`,
`isKnown`, `requiresArgument`, `allowsMultipleArguments`, `validateNoEmptyArgs`,
`validateSessionID`.

Kept: `List`/`entry` types (the map+order fields are visible — the structure is kept, the
semantics are excised), `IsEmpty`, `Get`, `Supports`, ALL capability name constants, error
vars (`ErrArgumentsRequired`, `ErrArguments`, `ErrMultipleArguments`, `ErrEmptyArgument`),
`userAgent` const, doc comments.

Tests deleted: `capability_test.go`, `list_test.go`, `list_fuzz_test.go` (the package's 3).
