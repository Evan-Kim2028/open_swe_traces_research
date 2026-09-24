# Exported API — iamsubj

Package `pkg/model/iam` (importable as `example.internal/clustkit/pkg/model/iam`).

- `type Subject interface { BuildAWSPolicy(*PolicyBuilder) (*Policy, error); ServiceAccount() (types.NamespacedName, bool) }` — an IAM identity.
- `func BuildNodeRoleSubject(igRole kops.InstanceGroupRole, enableLifecycleHookPermissions bool) (Subject, error)` — role→Subject factory.
- `func AddServiceAccountRole(context *IAMModelContext, podSpec *corev1.PodSpec, serviceAccountRole Subject) error` — mutates a pod spec with the token volume + env vars a service-account role needs.
- `type GenericServiceAccount{NamespacedName, Policy}`; node-role types `NodeRoleMaster`, `NodeRoleAPIServer`, `NodeRoleNode`, `NodeRoleBastion`.

Production callers: `pkg/model/iam/iam_builder.go` (subject enumeration), `pkg/model/components` / awsmodel pod-mutation paths.
