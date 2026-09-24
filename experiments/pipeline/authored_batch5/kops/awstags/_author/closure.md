# Closure — awstags

Package: `upup/pkg/fi/cloudup/awsup` (`example.internal/clustkit/upup/pkg/fi/cloudup/awsup`).

Files: `upup/pkg/fi/cloudup/awsup/aws_utils.go` (12 funcs).

Removed functions (bodies stubbed): `FindEC2Tag`, `FindASGTag`, `FindELBTag`, `FindELBV2Tag`,
`AWSErrorCode`, `AWSErrorMessage`, `EC2TagSpecification`, `ELBv2Tags`, `GetClusterName40`,
`GetResourceName32`, `NameForExternalTargetGroup`, `IsIAMNoSuchEntityException`.

Exported entry point(s): tag lookup/conversion used by all AWS task `Find`/`Render` code;
error classification used in retry/not-found paths.

Test files removed in excision: `aws_utils_test.go`.
