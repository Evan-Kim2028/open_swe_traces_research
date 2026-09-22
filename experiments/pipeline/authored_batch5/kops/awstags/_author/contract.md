# Contract — awstags

Tag lookup, AWS error unwrap, tag-spec conversion, and name helpers over the
aws-sdk-go-v2 types. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **Tag lookup.** Each `Find*Tag` helper scans its tag slice and returns the
   value of the first exact key match with `true`; an absent key returns
   `("", false)`; tags with nil keys or values are tolerated, with a nil value
   on a matching key yielding `""`. Covered by `TestDetail01`.
2. **Error unwrap.** `AWSErrorCode` and `AWSErrorMessage` recover the code and
   message of a `smithy.APIError` even when wrapped; errors that do not carry
   an API error produce `""`. Covered by `TestDetail02`.
3. **Tag specification.** `EC2TagSpecification` emits nothing for an empty or
   nil tag map and otherwise emits a single specification that carries every
   tag pair under the requested resource type. Covered by `TestDetail03`.
4. **Bounded names (shape).** `GetClusterName40` and `GetResourceName32`
   return deterministic, non-empty names no longer than their advertised
   limits, and the resource name carries the given prefix. The exact hash
   policy is an implementation detail and is not pinned. Covered by
   `TestDetail04`.
5. **Cluster sanitization.** `GetResourceName32` does not emit `.` characters;
   dots in the cluster name appear as `-` in the result. Covered by
   `TestDetail05`.
6. **Target-group ARN.** `NameForExternalTargetGroup` returns the middle
   segment of a `targetgroup/NAME/id` ARN resource and returns an error for
   malformed ARNs, non-`targetgroup` resources, and wrong segment counts.
   Covered by `TestDetail06`.
7. **Typed exception.** `IsIAMNoSuchEntityException` is true for a
   `NoSuchEntityException` even wrapped, and false for nil or unrelated
   errors. Covered by `TestDetail07`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes |
| TestDetail02 | 2 | partially |
| TestDetail03 | 3 | partially — asserts "no specs / one spec carrying all pairs"; nil-vs-empty not pinned |
| TestDetail04 | 4 | no — shape only (bounded, deterministic, prefix carried) |
| TestDetail05 | 5 | partially — asserts no dots and dot-to-dash mapping |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | yes |
