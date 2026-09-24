# Details — nodelabels

1. Role dispatch is exclusive-ordered: HasControlPlane > HasAPIServer > HasNode >
   HasBastion — a multi-role IG is classified by its first matching role in that order;
   an IG with none errors `unhandled instanceGroup role`. Inferable: partially — the
   precedence order is a choice.
2. Kubelet `NodeLabels` merge order: `ControlPlaneKubelet` (control-plane only) or
   `Kubelet`, then IG-level `Kubelet` — IG overrides cluster. Inferable: yes.
3. The apiserver role label is added for apiserver IGs unconditionally, and for
   control-plane IGs only when `featureflag.APIServerNodes` is enabled. Inferable:
   partially — the featureflag gate is a transitional quirk.
4. Role labels are set to the EMPTY string value (`"key": ""`). Inferable: no — empty vs
   "true" is arbitrary.
5. `Spec.NodeLabels` overlays everything — user labels can clobber role labels.
   Inferable: partially.
6. `nodeLabels` stays nil until a label is actually added — an all-empty input returns nil,
   not an empty map. Inferable: partially.
7. `BuildMandatoryControlPlaneLabels` mutates the passed map and returns it. Inferable: yes.
8. Label key spellings (`node-role.kubernetes.io/...`, clustkit-domain keys) are fixed
   strings. Inferable: no — exact spellings arbitrary.
