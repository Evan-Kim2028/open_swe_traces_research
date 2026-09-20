# Closure — iamsubj

Package: `pkg/model/iam` (`example.internal/clustkit/pkg/model/iam`).

Files: `pkg/model/iam/subject.go` (9 funcs).

Removed functions (bodies stubbed): `NodeRoleMaster.ServiceAccount`, `NodeRoleAPIServer.ServiceAccount`, `NodeRoleNode.ServiceAccount`, `NodeRoleBastion.ServiceAccount`, `GenericServiceAccount.ServiceAccount`, `GenericServiceAccount.BuildAWSPolicy`, `BuildNodeRoleSubject`, `AddServiceAccountRole`, `addServiceAccountRoleForAWS`.

Exported entry point(s): `BuildNodeRoleSubject` and `AddServiceAccountRole` — role→Subject dispatch and pod mutation for IRSA.
