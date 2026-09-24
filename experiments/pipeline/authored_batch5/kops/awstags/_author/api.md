# Exported API — awstags

Package `upup/pkg/fi/cloudup/awsup` (importable as `example.internal/clustkit/upup/pkg/fi/cloudup/awsup`).

AWS tag lookup/conversion, error-code extraction, and name truncation helpers.

- `FindEC2Tag`/`FindASGTag`/`FindELBTag`/`FindELBV2Tag` — linear scan over the SDK tag slice;
  `(value, true)` on first key match, `("", false)` otherwise.
- `AWSErrorCode`/`AWSErrorMessage` — `errors.As` into `smithy.APIError`; `""` for non-API
  errors.
- `EC2TagSpecification(resourceType, tags)` — `nil` for empty maps; else a single
  TagSpecification carrying all pairs.
- `ELBv2Tags(tags)` — `nil` for empty; else flat `[]elbv2types.Tag`.
- `GetClusterName40(cluster)` — truncates to 40 chars (hash added only when truncated).
- `GetResourceName32(cluster, prefix)` — `prefix + "-" + sanitized(cluster)` truncated to
  32, ALWAYS with a 6-char hash suffix.
- `NameForExternalTargetGroup(arn)` — parses `...:targetgroup/NAME/...` ARN, returns NAME;
  malformed ARNs or non-3-part/`targetgroup` resources error.
- `IsIAMNoSuchEntityException(err)` — `errors.As` `*iamtypes.NoSuchEntityException`.

Example: `GetResourceName32("my.cluster.example","lb")` → a ≤32-char `lb-...-hash` name.
