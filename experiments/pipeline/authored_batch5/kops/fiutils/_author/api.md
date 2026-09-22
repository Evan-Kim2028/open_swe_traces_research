# Exported API — fiutils

Package `upup/pkg/fi/utils` (importable as `example.internal/clustkit/upup/pkg/fi/utils`).

Assorted leaf helpers used across the `fi` framework.

- `func StringSlicesEqual(l, r []string) bool` — length + elementwise equality (order matters).
- `func StringSlicesEqualIgnoreOrder(l, r []string) bool` — multiset equality (order ignored).
- `func HashString(s string) (string, error)` — sha256 hex of the string.
- `func IsIPv6IP(s string) bool` — parses as an IP and does NOT map to v4.
- `func IsIPv4CIDR(s string) bool` — parses as a CIDR, maps to v4, and contains no `":"`.
- `func IsIPv6CIDR(s string) bool` — parses as a CIDR and does NOT map to v4.
- `func ParseCIDRNotation(subnet string) (int, int64, error)` — parses `/N#hexnum` into
  (newSize, netNum); anything else errors.
- `func CIDRSubnet(prefix string, newSize int, netNum int64) (string, error)` — the
  `netNum`-th subnet of `prefix` resized to `newSize` (big-endian numbering).
- `func SanitizeString(s string) string` — replaces chars outside
  `[a-zA-Z0-9_-]` with `_`, then keeps the LAST 200 chars.
- `func ExpandPath(p string) string` — a leading `~/` expands to the user's home dir;
  anything else passes through.
- `func YamlUnmarshal(yamlBytes []byte, dest interface{}) error` / `YamlMarshal` — YAML
  (un)marshal via `sigs.k8s.io/yaml` (JSON-compatible field names).
