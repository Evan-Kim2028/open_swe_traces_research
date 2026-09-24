# Details — podmutate

1. `imageRemapVisitor.VisitString` requires `path[-1]=="image"` AND `path[-3]` in
   {`containers`,`initContainers`} — depth < 3 or a different grandparent → skip with a
   warning. Inferable: partially — grandparent-position matching is a heuristic choice.
2. The mapper's result is written via the mutator ONLY when it differs. Inferable: yes.
3. `containerVisitor.VisitMap` fires on `path[-2]=="containers"` and `path[-1]` starting
   with `[` — so only container list ENTRIES, not the `containers` map/key itself.
   Inferable: partially.
4. `AddHostPathMapping` defaults mounts to `ReadOnly: true`; options are applied to the
   just-appended volume+mount. Inferable: yes (doc).
5. `WithReadWrite`/`WithType`/`WithHostPath`/`WithMountPath` mutate the mount vs volume
   appropriately (`WithType` sets `HostPath.Type` pointer). Inferable: yes.
6. `MarkPodAsCritical` APPENDS the toleration (existing tolerations preserved). Inferable: yes.
7. Priority markers overwrite `PriorityClassName` unconditionally. Inferable: yes.
8. `AddHostPathSELinuxContext` is gated on `cfg.ContainerdConfig.SeLinuxEnabled` — nil
   ContainerdConfig or disabled → no change. Sets `Type: "spc_t", Level: "s0"`. Inferable:
   no — the exact SELinux type/level strings are arbitrary.
