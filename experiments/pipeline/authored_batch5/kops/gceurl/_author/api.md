# Exported API — gceurl

Package `upup/pkg/fi/cloudup/gce` (importable as `example.internal/clustkit/upup/pkg/fi/cloudup/gce`).

GCE compute URL construction/parsing and RFC1035 label encoding.

- `GoogleCloudURL{Version, Project, Type, Name, Global, Region, Zone}`.
- `(u *GoogleCloudURL) BuildURL()` —
  `https://www.googleapis.com/compute/<version|v1>/` + optional `projects/P/`, `global/`,
  `regions/R/`, `zones/Z/` + `TYPE/NAME`.
- `ParseGoogleCloudURL(u)` — strict inverse: requires `https://www.googleapis.com/compute/`
  prefix, `v1`/`beta` version, `projects|zones|regions|global` path segments in any order,
  then `TYPE/NAME` as the final two tokens; anything trailing errors.
- `EncodeGCELabel(s)` — lowercase alnum bytes pass through; every other byte becomes
  `-XY` (two lowercase hex digits) — dash-prefixed hex escaping.
- `DecodeGCELabel(s)` — inverse: `-` → `%` then query-unescape; invalid escapes error.
- `TagForRole(clusterName, role)` —
  `ClusterPrefixedName("k8s-io-role-"+role.ToLowerString(), clusterName, 63)`.

Example: `EncodeGCELabel("a.b")` → `a-2eb`; `DecodeGCELabel("a-2eb")` → `a.b`.
