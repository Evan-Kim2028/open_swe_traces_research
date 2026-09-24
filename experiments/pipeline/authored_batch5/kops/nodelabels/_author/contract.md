# Contract — nodelabels

Node-label computation for instance groups in `pkg/nodelabels`. Every
commitment below is covered by a hidden test; every hidden test maps to a
commitment.

## Commitments

1. **Role dispatch.** A node IG gets the node role label; an apiserver IG
   gets the apiserver role label (and not the node label); a control-plane
   IG gets the control-plane label (and not the node label); a bastion IG
   gets none of them. An IG whose role matches no predicate errors (shape:
   a non-nil error). Role matching is plain equality against the constants,
   so a comma-list role is unhandled and errors. Covered by `TestDetail01`.
   *Deviation from DETAILS:* the "multi-role classified by first matching
   role" clause is not what the code does — a comma-list role matches no
   `HasX` predicate and errors like any unknown role. The test asserts the
   error, not the claimed precedence-on-lists.
2. **Kubelet label merge.** Node IGs merge `Spec.Kubelet.NodeLabels` then
   the IG's own `Kubelet.NodeLabels` (IG wins on conflict). Control-plane
   IGs merge `Spec.ControlPlaneKubelet` (never `Spec.Kubelet`) then the
   IG's `Kubelet.NodeLabels`. Covered by `TestDetail02`.
   *Deviation from api.md:* "(else `Spec.Kubelet`)" implies a fallback when
   `ControlPlaneKubelet` is nil; there is none — a control-plane IG with
   only `Spec.Kubelet` set merges nothing from it. The test asserts
   `Spec.Kubelet` is not consulted for control-plane.
3. **Apiserver role label gating.** Apiserver IGs always carry the
   apiserver role label; control-plane IGs carry it only when
   `featureflag.APIServerNodes` is enabled. Covered by `TestDetail03`.
4. **Empty-string role label values.** The node role label's value is the
   empty string, as shown in the api example. Covered by `TestDetail04`.
5. **User overlay.** `Spec.NodeLabels` are applied last and can overwrite
   generated role labels. Covered by `TestDetail05`.
6. **Nil on empty.** When nothing produces a label (e.g. a bare bastion
   IG), the returned map is nil, not an empty map. Covered by
   `TestDetail06`.
7. **Mandatory control-plane labels.** `BuildMandatoryControlPlaneLabels`
   mutates and returns the given map, adding the control-plane role label,
   the controller-PKI label, and the exclude-from-external-load-balancers
   label (three labels total). Covered by `TestDetail07`.
8. **Label key spellings (shape).** Mandatory-label keys are k8s-style
   qualified keys (contain `/`); exact spellings are not asserted.
   Covered by `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — per-role labels + error shape; multi-role clause deviates (errors) |
| TestDetail02 | 2 | yes — merge order; api.md fallback claim deviates (not asserted) |
| TestDetail03 | 3 | partially — flag gate asserted exactly |
| TestDetail04 | 4 | no — empty value documented by api example, asserted |
| TestDetail05 | 5 | partially — clobbering asserted |
| TestDetail06 | 6 | partially — nil-not-empty asserted |
| TestDetail07 | 7 | yes — mutate+return, three added labels |
| TestDetail08 | 8 | no — key shape (`/` qualified) only |
