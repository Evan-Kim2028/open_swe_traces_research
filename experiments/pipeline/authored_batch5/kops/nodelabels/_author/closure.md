# Closure — nodelabels

Package: `pkg/nodelabels` (`example.internal/clustkit/pkg/nodelabels`).

Files: `pkg/nodelabels/builder.go` (2 funcs).

Removed functions (bodies stubbed): `BuildNodeLabels`,
`BuildMandatoryControlPlaneLabels`.

Exported entry point(s): `BuildNodeLabels` — used by the labels controller and cluster
apply paths to compute kubelet node labels.

Depends on `InstanceGroupRole.Has*` predicates (kept — owned by the `igroles` unit only in
`instancegroup.go`; these predicates live there but are NOT excised here).

Test files removed in excision: `builder_test.go`.
