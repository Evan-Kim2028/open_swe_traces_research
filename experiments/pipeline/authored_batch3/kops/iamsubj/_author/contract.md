# Contract (L2) — iamsubj

`ServiceAccount` on all four node-role types returns empty+false — node roles have no k8s identity. `GenericServiceAccount` echoes its NamespacedName+true and returns its stored Policy from `BuildAWSPolicy`. `BuildNodeRoleSubject` maps each `InstanceGroupRole` to its type — ControlPlane→Master, APIServer→APIServer (flag→`warmPool`), Node→Node (flag→`enableLifecycleHookPermissions`), Bastion→Bastion — and errors on unknown roles. `AddServiceAccountRole` proceeds only on AWS. `addServiceAccountRoleForAWS` builds `arn:<partition>:iam::<acct>:role/<name>`, adds a projected `token-amazonaws-com` volume (0644, audience `amazonaws.com`, 86400s, path `token`), mounts `/var/run/secrets/amazonaws.com/` read-only in every container plus `AWS_ROLE_ARN`/`AWS_WEB_IDENTITY_TOKEN_FILE` env, and sets FSGroup=`wellknownusers.Generic` only when unset.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRoundTrip` | policy build subjects round-trip through the builder |
| `TestPolicyGeneration`, `TestEmptyPolicy`, `TestAsJSONIsIdempotent` | subject→policy pipeline (BuildAWSPolicy dispatch) |
| `TestAddKMSIAMPolicies`, `TestKmsViaServices`, `TestIAMServiceEC2`, `TestAddKarpenterPermissions` | per-subject policy statements via the builder |
