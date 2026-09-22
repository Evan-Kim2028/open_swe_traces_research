# Details — igroles

1. Each `InstanceGroupRole.HasX()` is a plain equality check against the matching
   `InstanceGroupRoleX` constant — no aliasing, no case folding. Inferable: yes — the
   constant-per-predicate shape is the only reading consistent with the type.
2. `IsControlPlaneType` ORs HasControlPlane/HasAPIServer/HasEtcd/HasScheduler/
   HasKubeControllerManager — Node and Bastion are NOT control-plane types. Inferable:
   partially — the membership list is a policy choice, though the name constrains it.
3. `ToLowerString` returns the literal `"control-plane"` (with hyphen) for the control-plane
   role and `strings.ToLower(string(r))` otherwise — `"APIServer"` → `"apiserver"`, not
   `"api-server"`. Inferable: partially — the hyphenated special case is arbitrary.
4. `RunsAPIServer`/`RunsEtcd`/`RunsScheduler`/`RunsKubeControllerManager` are
   `IsControlPlane() || IsXOnly()` — control-plane groups report running all four daemons.
   Inferable: yes — control-plane nodes run the control plane daemons by definition.
5. `IsControlPlane`/`IsAPIServerOnly`/`IsBastion` are single-role equality; there is no
   wildcard or empty-role fallback. Inferable: yes.
6. `HasGVisor` is a conjunction: role must be Node AND `Spec.Containerd.GVisor.Enabled`
   must be a non-nil true; any missing level is false, not panic. Inferable: partially —
   the Node restriction is a policy choice (doc comment hints it).
7. `IsKarpenterManaged` requires `Spec.Manager == InstanceManagerKarpenter` AND role Node —
   a Karpenter-managed control-plane group returns false. Inferable: partially.
8. `AddInstanceGroupNodeLabel` writes key `"kops.k8s.io/instancegroup"` with the group's
   `Name` as value and allocates `Spec.NodeLabels` when nil. Inferable: no — the label
   key string is arbitrary.
