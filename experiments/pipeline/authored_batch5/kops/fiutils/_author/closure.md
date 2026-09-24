# Closure — fiutils

Package: `upup/pkg/fi/utils` (`example.internal/clustkit/upup/pkg/fi/utils`).

Files: `equals.go` (2 funcs), `hash.go` (1), `net.go` (5), `sanitize.go` (2), `yaml.go` (2) —
12 funcs.

Removed functions (bodies stubbed): `StringSlicesEqual`, `StringSlicesEqualIgnoreOrder`,
`HashString`, `IsIPv6IP`, `IsIPv4CIDR`, `IsIPv6CIDR`, `ParseCIDRNotation`, `CIDRSubnet`,
`SanitizeString`, `ExpandPath`, `YamlUnmarshal`, `YamlMarshal`.

Exported entry point(s): the whole file set — leaf helpers used by task `Find`/`Run`
comparison, naming, and YAML loading throughout `fi`.

Test files removed in excision: `equals_test.go`, `hash_test.go`, `sanitize_test.go`.
