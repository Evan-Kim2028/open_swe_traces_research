# Contract — kapiutil

Label/taint/version helpers in `pkg/apis/kops/util`. Every commitment below
is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **`GetNodeRole`** inspects `node-role.kubernetes.io/{master,
   control-plane, node, api-server}` in that order and returns the matching
   short name (`master`/`control-plane`/`node`/`apiserver`); with none
   present it falls back to `kubernetes.io/role`, else `""`. Covered by
   `TestDetail01`.
2. **`ParseTaint`** splits on `:` — one part is a bare key (a `k=v` single
   part keeps `k=v` as the key); two parts are `key[=value]:effect`. The
   returned map always contains `key`, `value`, and `effect` (empty strings
   when absent). More than two colon-separated parts or more than one `=` is
   an error. Covered by `TestDetail02`.
3. **`ParseKubernetesVersion`** uses tolerant semver first, then the
   `/v1.<minor>.` URL form synthesizing a `1.<minor>` version; everything
   else errors. Covered by `TestDetail03`.
4. **`IsKubernetesGTE`** panics when the `version` argument won't parse and
   strips Pre/Build from `k8sVersion` before comparing. Covered by
   `TestDetail04`.
5. **`ParseVersion`** is strict semver — `"1.2"` fails where the tolerant
   parser succeeds; `String`/`IsInRange` accessors behave. Covered by
   `TestDetail05`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — order asserted per api.md |
| TestDetail02 | 2 | partially — map shape + always-present keys per api.md |
| TestDetail03 | 3 | partially — URL fallback asserted per api.md |
| TestDetail04 | 4 | partially — panic asserted; Pre/Build strip asserted |
| TestDetail05 | 5 | yes |
