# Bug report

IAM subject handling is broken: instance-group roles build the wrong subject kind (or panic on valid roles), node subjects claim to have kubernetes service accounts, and pods destined to run under a service-account IAM role get no credentials — the projected token volume, environment variables, and file group the cloud SDK expects are all missing.

Expected: each instance-group role maps to its own subject type and an unknown role is an error; node-role subjects report no service account; an AWS service-account role adds a projected token volume, the role-ARN and token-file environment variables on every container, and a default file group so the token is readable; non-AWS clouds are rejected.

Reproduce with:

```
go test -count=1 ./pkg/model/iam/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
