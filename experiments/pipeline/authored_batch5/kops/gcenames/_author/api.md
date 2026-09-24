# Exported API — gcenames

Package `upup/pkg/fi/cloudup/gce` (importable as `example.internal/clustkit/upup/pkg/fi/cloudup/gce`).

GCE naming + error helpers.

- `IsNotFound(err)` — true iff `errors.As` yields a `googleapi.Error` with `Code == 404`.
- `IsNotReady(err)` — true iff a `googleapi.Error` has an `Errors[]` entry with
  `Reason == "resourceNotReady"`.
- `ClusterPrefixedName(objectName, clusterName, maxLength)` — `safeCluster-objectName`
  truncated to `maxLength` (dots → dashes; hash-suffix truncation; fatal if suffix leaves
  <10 chars).
- `ClusterSuffixedName(objectName, clusterName, maxLength)` — `objectName-safeCluster`
  truncated to `maxLength` (same rules).
- `SafeClusterName(clusterName)` — dots → dashes.
- `LabelForCluster(clusterName)` — `Label{Key: <cluster label>, Value: safeName}`.
- `SafeTruncatedClusterName(clusterName, maxLength)` — safe name + truncate.
- `SafeObjectName(name, clusterName)` — `name-clusterName` made safe.
- `ServiceAccountName(name, clusterName)` — `ClusterSuffixedName` at length 30.
- `LastComponent(s)` — substring after the last `/` (whole string if none).
- `SSHUsernameForImage(image)` — `"ubuntu"` when the image's last component starts with
  `ubuntu` (any case); else the primary SSH secret name (`"admin"`).
- `ZoneToRegion(zone)` — `a-b-c` → `a-b`; zones with ≤2 `-` segments error.

Example: `ClusterPrefixedName("https","cluster.example.com",38)` → `cluster-example-com-https`.
