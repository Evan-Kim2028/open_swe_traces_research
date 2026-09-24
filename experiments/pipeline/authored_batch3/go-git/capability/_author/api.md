# Exported API — capability

Package `plumbing/protocol/capability` (module `example.internal/gitkit/v6`).

`List` — insertion-ordered capability set (zero value usable). Methods:
`IsEmpty`, `Get(name) []string`, `Set(name, values...)`, `Add(name, values...)`,
`Supports(name) bool`, `Delete(name)`, `All() []string`, `MarshalText/AppendText`,
`UnmarshalText`, `String`. Package functions `DecodeList(raw []byte, l *List)`,
`EncodeList(l *List) []byte`, `Validate(l *List) error`, `DefaultAgent() string`.

All `Capability` constants stay visible (MultiACK … SessionID); error vars
`ErrArgumentsRequired`, `ErrArguments`, `ErrMultipleArguments`, `ErrEmptyArgument` stay.

Callers: every packp message carries a `capability.List`; transports validate before send.
In-tree tests removed: the 3 in this package.
