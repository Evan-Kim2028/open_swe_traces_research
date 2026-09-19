# Closure — clustervalid

Package: pkg/validation. Files: validate_cluster.go, node_conditions.go.

Removed: 9 functions (bodies stubbed to `panic("excised: <name>")`, signatures and doc comments preserved, compiles clean).

`NewClusterValidator(...) (ClusterValidator, error)`; `(clusterValidatorImpl) Validate(ctx) (*ValidationCluster, error)`; `(ValidationCluster) validateNodes(...)`; `(ValidationCluster) collectPodFailures(...)`; `(ValidationCluster) addError`; `hasPlaceHolderIP`; `isNodeReady`; `getNodeReadyStatus`; `findNodeCondition`.
