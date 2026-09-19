# Contract (L2) — clustervalid

Cluster validation walks cloud instance groups plus the live node/pod lists and records failures without aborting the scan (except DNS/API-host lookup errors). If the cluster publishes DNS records, the API host is resolved; a placeholder IPv4/IPv6 address is a single DNS failure and the rest of validation is skipped. Otherwise nodes are listed and mapped to cloud groups.

Instance groups: a group absent from the cloud is a failure. Members in detached state do not count toward the current size and are not required to have joined. Warm-pool members and bastion-role members are not expected to join the node list. Current size (non-detached) below the cloud target size is a failure. A machine that should have joined but has no node object is a failure, unless that id is in the tolerated-unready set.

Readiness: a node is ready only if its Ready condition is True and NetworkUnavailable is not True (missing NetworkUnavailable is allowed). Control-plane, apiserver, and node roles that are not ready are failures unless tolerated; other roles are ignored. Ready nodes are returned for the pod pass. Worker unready count: if max-unready > 0 and the number of non-detached, non-warm-pool unready workers is in (0, max], those names/ids are tolerated for both node-not-ready and critical-pod checks.

Pods: only system-cluster-critical and system-node-critical priorities are considered, after the caller’s pod filter. Succeeded pods are ignored. Pending or unknown-phase pods fail; containers that are not Ready fail (message lists those container names). system-node-critical failures attach the node’s instance group; cluster-critical ones do not. Pods on a tolerated node are skipped. Each ready control-plane node (label `node-role.kubernetes.io/control-plane`) must run kube-apiserver, kube-controller-manager, and kube-scheduler in kube-system (`k8s-app`); a missing static pod is a Node failure.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `Test_ValidateCloudGroupMissing` | instance group absent from the cloud is a failure |
| `Test_ValidateNodesNotEnough` | non-detached size below target is a failure |
| `Test_ValidateDetachedNodesDontCount` | detached members do not count toward size |
| `Test_ValidateDetachedNodesNotValidated` | detached members are not required to join or be ready |
| `Test_ValidateNodeNotReady` | unready worker/node role is a failure |
| `Test_ValidateMastersNotEnough` | control-plane group undersized vs target |
| `Test_ValidateMasterNotReady` | unready control-plane node is a failure |
| `Test_ValidateMasterStaticPods` | missing kube-apiserver/controller-manager/scheduler on a ready control-plane node |
| `Test_ValidateNoPodFailures` | healthy critical pods produce no pod failures |
| `Test_ValidatePodFailure` | pending/unknown/unready critical pods are failures |
| `Test_ValidateBastionNodes` | bastion role is not expected to join |
| `Test_ValidateAllowedNotReadyNodes` | at most N unready workers are tolerated for node and pod checks |

Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/validation/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
