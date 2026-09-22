# Bug report

AWS tag/error/name helpers are broken: the `Find*Tag` lookups return not-found for present
keys (or match the wrong tag shape), `AWSErrorCode`/`AWSErrorMessage` don't unwrap
`smithy.APIError`, `EC2TagSpecification`/`ELBv2Tags` mis-handle empty maps and tag structure,
`GetClusterName40`/`GetResourceName32` violate their length caps or hash policy,
`NameForExternalTargetGroup` extracts the wrong ARN segment, and
`IsIAMNoSuchEntityException` doesn't unwrap.

Expected: exact-key first-match lookups; `errors.As` extraction of code/message; nil for
empty tag maps; 40-char cluster cap and always-hashed 32-char resource names; middle-segment
targetgroup name; typed-exception detection.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/cloudup/awsup/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
