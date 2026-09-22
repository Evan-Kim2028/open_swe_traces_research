# Exported API — kapiutil

Package `pkg/apis/kops/util` (importable as `example.internal/clustkit/pkg/apis/kops/util`).

Small label/taint/version helpers.

- `func GetNodeRole(node *v1.Node) string` — inspects `node-role.kubernetes.io/*` labels in the
  order master, control-plane, node, api-server → returns `"master"`, `"control-plane"`,
  `"node"`, `"apiserver"`; falls back to `Labels["kubernetes.io/role"]` when none present.
- `func ParseTaint(st string) (map[string]string, error)` — parses `key[=value]:effect` into a
  map with keys `key`, `value`, `effect` (all always present, empty string when absent).
  `">2` colon-separated parts or `>2` `=`-parts is an `invalid taint spec` error.
- `func ParseKubernetesVersion(version string) (*semver.Version, error)` — tolerant semver
  parse; on failure tries a `/v1\.(\d+)\.` URL pattern and synthesises `{Major:1, Minor:n}`.
- `func IsKubernetesGTE(version string, k8sVersion semver.Version) bool` — deprecated helper:
  parses `version` (PANICS on error), strips Pre/Build from `k8sVersion`, returns GTE.
- `func ParseVersion(s string) (*Version, error)` — strict `semver.Parse` into the `Version`
  wrapper; `(*Version).String()` and `(*Version).IsInRange(range)` accessors.
