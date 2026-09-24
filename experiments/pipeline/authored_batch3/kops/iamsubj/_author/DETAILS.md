# Details — iamsubj

1. `ServiceAccount` on all four node-role types returns an empty NamespacedName + false — node roles have no kubernetes identity. Inferable: yes — stated on the interface doc.
2. `GenericServiceAccount.ServiceAccount` echoes its NamespacedName + true; `BuildAWSPolicy` returns its stored Policy verbatim. Inferable: yes.
3. `BuildNodeRoleSubject` maps ControlPlane→NodeRoleMaster, APIServer→NodeRoleAPIServer (carrying the flag as `warmPool`), Node→NodeRoleNode (carrying `enableLifecycleHookPermissions`), Bastion→NodeRoleBastion; unknown roles error. Inferable: partially — the asymmetric field mapping is a choice.
4. `AddServiceAccountRole` dispatches on `cluster.GetCloudProvider()` — only AWS proceeds; anything else errors. Inferable: partially.
5. `addServiceAccountRoleForAWS` builds `arn:<partition>:iam::<account>:role/<name>`; adds ONE projected volume (`token-amazonaws-com`, mode 0644) containing a ServiceAccountToken (audience `amazonaws.com`, 86400s expiry, path `token`). Inferable: no — the literal spellings are arbitrary.
6. Every container gains a read-only mount at `/var/run/secrets/amazonaws.com/` plus env `AWS_ROLE_ARN` and `AWS_WEB_IDENTITY_TOKEN_FILE` pointing at the token. Inferable: no — env/volume spellings.
7. The pod's SecurityContext gains FSGroup=`wellknownusers.Generic` — created if nil, left alone if already set. Inferable: no.
