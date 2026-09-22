# Closure — podmutate

Package: `pkg/kubemanifest` (`example.internal/clustkit/pkg/kubemanifest`).

Files: `images.go` (2), `containerargs.go` (2), `volumes.go` (5), `critical.go` (1),
`priority.go` (2), `selinux.go` (1) — 13 funcs.

Removed functions (bodies stubbed): `Object.RemapImages`, `imageRemapVisitor.VisitString`,
`Object.VisitContainers`, `containerVisitor.VisitMap`, `AddHostPathMapping`, `WithReadWrite`,
`WithType`, `WithHostPath`, `WithMountPath`, `MarkPodAsCritical`, `MarkPodAsNodeCritical`,
`MarkPodAsClusterCritical`, `AddHostPathSELinuxContext`.

Exported entry point(s): `RemapImages` (used by asset remapping) and `MarkPodAs*` /
`AddHostPathMapping` (used by nodeup/bootstrap manifests). Depends on `Object.accept`/`visit`
from `manifest`/`visitor` (separate unit).

Test files removed in excision: none.
