# Exported API — clustervalid

`NewClusterValidator(cluster, cloud, instanceGroupList, filterInstanceGroups, filterPodsForValidation, maxUnreadyNodes, restConfig, k8sClient) (ClusterValidator, error)`

`ClusterValidator.Validate(ctx) (*ValidationCluster, error)` returns a result with `Failures []*ValidationError` and `Nodes []*ValidationNode`.

A missing instance-group list is an error. Nil filters mean “validate everything.”

Callers: cluster-validate CLI / rolling-update gates (`cmd/kops/validate_cluster.go` and friends). In-tree tests construct the validator and call `Validate`.
