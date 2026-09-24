# Contract — gcenames

GCE naming and error-classification helpers in `upup/pkg/fi/cloudup/gce`.
Every commitment below is covered by a hidden test; every hidden test maps
to a commitment.

## Commitments

1. **Error classifiers.** `IsNotFound` unwraps (`errors.As`) and matches a
   `googleapi.Error` with `Code == 404`; `IsNotReady` does NOT unwrap and
   scans `Errors[].Reason` for `resourceNotReady`. Covered by
   `TestDetail01`.
2. **Dot-free cluster names.** All name helpers replace `.` with `-` in the
   cluster name (`SafeClusterName`, `SafeObjectName`, and the composed
   helpers). Covered by `TestDetail02`.
3. **Prefixed/suffixed composition (shape).** `ClusterPrefixedName` is
   `safeCluster-objectName` truncated to `maxLength` (the object suffix is
   preserved and a hash makes truncated names input-sensitive);
   `ClusterSuffixedName` is `objectName-safeCluster` likewise. When the
   fixed part leaves too little room the helpers fail fatally — asserted as
   process death in a child process; the 10-char floor and the hash length
   are not pinned. Covered by `TestDetail03`.
4. **`LabelForCluster` (shape).** Returns the package's cluster label key
   with the safe cluster name as value. The key spelling is the package
   constant, not a literal in the test. Covered by `TestDetail04`.
5. **`ServiceAccountName` (shape).** Equals `ClusterSuffixedName` applied at
   the documented 30-char cap; output never exceeds the cap. Covered by
   `TestDetail05`.
6. **`LastComponent`** returns the substring after the last `/`, or the
   whole string when there is none. Covered by `TestDetail06`.
7. **`SSHUsernameForImage`.** `ubuntu` when the image's last path component
   starts with `ubuntu` (any case); otherwise the primary SSH username
   `admin`. Covered by `TestDetail07`.
8. **`ZoneToRegion`** drops the last dash-segment (`a-b-c` → `a-b`); zones
   with ≤2 segments error. Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — unwrap asymmetry asserted per the row |
| TestDetail02 | 2 | doc |
| TestDetail03 | 3 | no — shape: truncation bounds + determinism + death; constants not pinned |
| TestDetail04 | 4 | no — shape: package constant key + safe-name value |
| TestDetail05 | 5 | no — asserted via ClusterSuffixedName equivalence per api.md |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | partially — ubuntu prefix rule and "admin" fallback per doc comment |
| TestDetail08 | 8 | yes |
