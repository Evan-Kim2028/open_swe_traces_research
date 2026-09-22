# Details — kapiutil

1. `GetNodeRole` checks the four `node-role.kubernetes.io/` keys in a FIXED order
   (master → control-plane → node → api-server) and returns the matching short name;
   a node carrying both `master` and `node` labels reports `"master"`. Inferable:
   partially — the precedence order is a choice.
2. `ParseTaint` splits on `:` — 1 part = key only; 2 parts = `key[=value]:effect` — and
   always returns a map with `key`/`value`/`effect` keys present (empty strings when
   absent). `a:b:c` or `k=a=b:e` is an error. Inferable: partially — the map-of-strings
   shape and always-present keys are choices.
3. `ParseKubernetesVersion` uses `semver.ParseTolerant` first, then the URL regex
   `/v1\.(\d+)\.` → `1.<minor>`; everything else errors. Inferable: partially — the URL
   fallback pattern is arbitrary.
4. `IsKubernetesGTE` PANICS when `version` won't parse (no error return), and strips
   Pre/Build from the `k8sVersion` argument before comparing. Inferable: partially —
   panic-on-bad-input is a choice.
5. `ParseVersion` uses STRICT `semver.Parse` (not tolerant) — `"1.2"` fails where
   `ParseKubernetesVersion("1.2")` succeeds. Inferable: yes — the two parsers visibly
   differ in the same file.
