# Exported API — fieldmap

Package `pkg/apis/kops` (importable as `example.internal/clustkit/pkg/apis/kops`).

- `func HumanPathForClusterField(fieldPath string) string` — the path to print for users for a given cluster field.
- `func InternalPathForClusterField(fieldPath string) string` — the path as it appears in the internal API.
- `func NewClusterField(path string) *ClusterField`, and methods `HumanPath`, `InternalPath`, `PathInV1Alpha2`, `PathInV1Alpha3` on `*ClusterField`.

Production callers: `pkg/commands/set_cluster.go`, `upup/pkg/fi/cloudup/metal/cloud.go`.
