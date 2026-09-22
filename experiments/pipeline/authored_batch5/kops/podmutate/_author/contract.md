# Contract — podmutate

Pod/manifest mutators in `pkg/kubemanifest` built on the `Object` visitor.
Every commitment below is covered by a hidden test; every hidden test maps
to a commitment.

## Commitments

1. **Image remap scope.** `RemapImages` applies the mapper only to string
   fields whose last path element is `image` and whose grandparent element
   is `containers` or `initContainers`; a top-level `image` key (depth < 3)
   and an `image` under any other grandparent are skipped. The mapper is
   invoked exactly on the qualifying fields. Covered by `TestDetail01`.
2. **Write only on change.** The mapper's result is applied to the field;
   an identity mapper leaves values unchanged and still sees each
   qualifying field once. (The write-through-mutator-only-on-diff path is
   not externally observable; the observable half — mapper sees exactly the
   qualifying fields and changed results land — is asserted.) Covered by
   `TestDetail02`.
3. **Container entry visits.** `VisitContainers` fires for each map at a
   slice index (`[i]`) under a `containers` path element — including nested
   and top-level occurrences — and never for the containers collection
   itself or maps under a non-slice `containers` key. Covered by
   `TestDetail03`.
4. **HostPath mapping defaults.** `AddHostPathMapping` appends (does not
   replace) a `HostPath` volume and a volume mount; the mount defaults to
   read-only at the given path. Covered by `TestDetail04`.
5. **Options.** `WithReadWrite` clears the mount's `ReadOnly`; `WithType`
   sets the volume's `HostPath.Type` pointer; `WithHostPath`/`WithMountPath`
   retarget the volume path and mount path respectively. Covered by
   `TestDetail05`.
6. **Critical toleration append.** `MarkPodAsCritical` appends a
   `CriticalAddonsOnly`/`Exists` toleration, preserving existing ones.
   Covered by `TestDetail06`.
7. **Priority overwrite.** `MarkPodAsNodeCritical`/`MarkPodAsClusterCritical`
   overwrite `PriorityClassName` unconditionally; the two priorities differ.
   (Literal class names not pinned.) Covered by `TestDetail07`.
8. **SELinux gate.** `AddHostPathSELinuxContext` is a no-op when
   `cfg.ContainerdConfig` is nil or `SeLinuxEnabled` is false; when enabled
   it sets the documented pod-level `SELinuxOptions` (`spc_t`/`s0`),
   creating `SecurityContext` if needed. Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — scope predicate asserted exactly |
| TestDetail02 | 2 | yes — observable half asserted (mapper scope + application) |
| TestDetail03 | 3 | partially — slice-index-only visitation asserted |
| TestDetail04 | 4 | yes (doc) — append + read-only default |
| TestDetail05 | 5 | yes — each option's effect asserted |
| TestDetail06 | 6 | yes — append preserves existing tolerations |
| TestDetail07 | 7 | yes — overwrite + non-empty + distinct priorities |
| TestDetail08 | 8 | no — gate asserted; spc_t/s0 documented in api.md, asserted |
