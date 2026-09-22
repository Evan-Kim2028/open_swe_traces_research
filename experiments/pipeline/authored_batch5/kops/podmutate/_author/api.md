# Exported API — podmutate

Package `pkg/kubemanifest` (importable as `example.internal/clustkit/pkg/kubemanifest`).

Manifest/pod mutators built on the `Object` visitor.

- `Object.RemapImages(mapper ImageRemapFunction)` — visits every string field; a field whose
  last path element is `image` AND whose grandparent element is `containers` or
  `initContainers` gets `mapper(v)` applied (written back only when different). Other
  `image`-looking fields are warned and skipped.
- `Object.VisitContainers(visitorFn)` — visits every map whose parent path element is
  `containers` and whose own key is a slice index (`[i]`) — i.e. each container entry.
- `AddHostPathMapping(pod, container, name, path, options...)` — appends a `HostPath` volume
  plus a READ-ONLY `VolumeMount`, then applies `HostPathMappingOption`s.
- Options: `WithReadWrite()` (clears ReadOnly), `WithType(t)`, `WithHostPath(p)`,
  `WithMountPath(p)`.
- `MarkPodAsCritical(pod)` — appends a `CriticalAddonsOnly`/`Exists` toleration.
- `MarkPodAsNodeCritical`/`MarkPodAsClusterCritical` — set `PriorityClassName`.
- `AddHostPathSELinuxContext(pod, cfg)` — no-op unless `cfg.ContainerdConfig.SeLinuxEnabled`;
  then sets pod-level `SELinuxOptions` `spc_t`/`s0` (creates `SecurityContext` if needed).

Example: `RemapImages` on a Deployment rewrites `spec.template.spec.containers.[0].image`
but leaves a top-level `image:` key untouched.
