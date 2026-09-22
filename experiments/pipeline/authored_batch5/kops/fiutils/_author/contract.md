# Contract — fiutils

Assorted leaf helpers in `upup/pkg/fi/utils`. Every commitment below is
covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **Slice equality.** `StringSlicesEqual` is order- and length-sensitive;
   `StringSlicesEqualIgnoreOrder` compares as a multiset and sorts a copy
   (inputs are not mutated). Covered by `TestDetail01`.
2. **`HashString`** returns the lowercase hex sha256 of the input (64 chars).
   Covered by `TestDetail02` (compared against `crypto/sha256`).
3. **IP/CIDR classification.** `IsIPv6IP`/`IsIPv6CIDR` are false for
   v4-mappable values; `IsIPv4CIDR` additionally rejects any string
   containing `":"`, so a v4-mappable v6 CIDR is neither. Covered by
   `TestDetail03`.
4. **`ParseCIDRNotation` (shape).** Parses the documented `/N#hexnum` form
   into `(newSize, netNum)`; malformed input errors. Acceptance rules beyond
   the documented form are not asserted. Covered by `TestDetail04`.
5. **`CIDRSubnet`** returns the `netNum`-th subnet of the prefix resized to
   `newSize`, big-endian subnet numbering (`netNum` is an index). Covered by
   `TestDetail05`.
6. **`SanitizeString`.** Replaces disallowed characters rather than removing
   them and keeps only a bounded tail of a long input; per the API doc the
   allowed set is `[a-zA-Z0-9_-]`, the replacement is `_`, and the bound is
   the last 200 chars. Covered by `TestDetail06`. (DETAILS marks the
   charset/replacement/length arbitrary — the assertions match the API doc
   exactly and assert nothing beyond it.)
7. **`ExpandPath`.** Expands a leading `~/` via the user's home directory;
   a bare `~` and all other inputs pass through unchanged. Covered by
   `TestDetail07`.
8. **YAML via JSON tags.** `YamlUnmarshal`/`YamlMarshal` wrap
   `sigs.k8s.io/yaml`, so `json` field tags apply. Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | yes |
| TestDetail03 | 3 | partially — the `:` rejection asserted per api.md |
| TestDetail04 | 4 | no — shape only (documented form parses, garbage errors) |
| TestDetail05 | 5 | partially — index semantics asserted, edge errors shape-only |
| TestDetail06 | 6 | no — asserted per api.md (charset, `_`, last-200); nothing extra |
| TestDetail07 | 7 | partially — `~/` expansion and pass-through asserted |
| TestDetail08 | 8 | yes |
