# Closure — kapiutil

Package: `pkg/apis/kops/util` (`example.internal/clustkit/pkg/apis/kops/util`).

Files: `pkg/apis/kops/util/labels.go` (1 func), `taints.go` (1), `versions.go` (5) — 7 funcs.

Removed functions (bodies stubbed): `GetNodeRole`, `ParseTaint`, `ParseKubernetesVersion`,
`IsKubernetesGTE`, `ParseVersion`, `Version.String`, `Version.IsInRange`.

Exported entry point(s): `ParseKubernetesVersion`/`IsKubernetesGTE` are used by `cluster.go`
predicates and upgrade logic; `GetNodeRole`/`ParseTaint` by node inspection and taint handling.

Test files removed in excision: `labels_test.go`, `taints_test.go`, `versions_test.go` (all
cover excised funcs directly).
