# Exported API — nodelabels

Package `pkg/nodelabels` (importable as `example.internal/clustkit/pkg/nodelabels`).

Node-label computation for instance groups (the kubelet labels moved to a controller in
k8s 1.16).

- `BuildNodeLabels(cluster, instanceGroup)` — label map for the IG:
  - Control-plane IGs merge `cluster.Spec.ControlPlaneKubelet` (else `Spec.Kubelet`) then the
    IG's own `Kubelet.NodeLabels`.
  - Adds `node-role.kubernetes.io/api-server` for apiserver/control-plane IGs (gated on
    `featureflag.APIServerNodes` for pure control-plane).
  - Adds `node-role.kubernetes.io/node` for node IGs.
  - Control-plane IGs also get the mandatory control-plane labels.
  - `instanceGroup.Spec.NodeLabels` overlay last (user labels win).
  - Bastion IGs get only merged kubelet/spec labels; unknown roles error.
- `BuildMandatoryControlPlaneLabels(nodeLabels)` — sets
  `node-role.kubernetes.io/control-plane`, the controller-PKI label, and
  `node.kubernetes.io/exclude-from-external-load-balancers` on the given map.

Example: a node IG with `Spec.NodeLabels{"a":"b"}` → `{"a":"b",
"node-role.kubernetes.io/node":""}`.
