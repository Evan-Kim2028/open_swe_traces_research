# Closure — gcenames

Package: `upup/pkg/fi/cloudup/gce` (`example.internal/clustkit/upup/pkg/fi/cloudup/gce`).

Files: `upup/pkg/fi/cloudup/gce/utils.go` (12 funcs).

Removed functions (bodies stubbed): `IsNotFound`, `IsNotReady`, `ClusterPrefixedName`,
`ClusterSuffixedName`, `SafeClusterName`, `LabelForCluster`, `SafeTruncatedClusterName`,
`SafeObjectName`, `ServiceAccountName`, `LastComponent`, `SSHUsernameForImage`,
`ZoneToRegion`.

Exported entry point(s): name construction used by every GCE task (instances, firewalls,
service accounts); error classification used by `Find` polling.

Test files removed in excision: `utils_test.go`.
