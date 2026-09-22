# Bug report

Pod-manifest mutators are broken: `RemapImages` rewrites every `image` key (including
non-container ones) or none at all, `VisitContainers` never fires on container entries,
`AddHostPathMapping` mounts read-write or doesn't apply options, `MarkPodAsCritical` replaces
rather than appends tolerations, the priority markers set the wrong class, and
`AddHostPathSELinuxContext` applies the context even when SELinux is disabled.

Expected: image remap only under `containers`/`initContainers`; container-entry visits;
read-only default with option overrides; toleration append; `system-node-critical` /
`system-cluster-critical`; SELinux context only when enabled.

Reproduce with:

```
go test -count=1 ./pkg/kubemanifest/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
