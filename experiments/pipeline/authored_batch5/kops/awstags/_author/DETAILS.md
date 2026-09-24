# Details — awstags

1. The four `Find*Tag` functions share semantics: first exact key match wins; absent key →
   `("", false)`. Nil keys/values deref via `aws.ToString` (nil-safe). Inferable: yes.
2. `AWSErrorCode`/`AWSErrorMessage` unwrap via `errors.As` to `smithy.APIError` — a wrapped
   API error still yields its code; non-API errors give `""` (not the raw `err.Error()`).
   Inferable: partially.
3. `EC2TagSpecification` returns nil (not an empty slice/spec) for an empty tag map; the
   single spec groups ALL tags under the given resource type. Inferable: partially — nil
   vs empty is a choice.
4. `GetClusterName40` truncates with `AlwaysAddHash:false` — short names pass through
   unmodified. `GetResourceName32` sets `AlwaysAddHash:true` — even short results carry a
   hash. Inferable: no — the asymmetric hash policy is arbitrary.
5. `GetResourceName32` also replaces `.` with `-` in the cluster part BEFORE prefixing.
   Inferable: partially.
6. `NameForExternalTargetGroup` splits the ARN `Resource` on `/` and demands exactly
   `targetgroup/NAME/id` (3 parts, literal `targetgroup` head); it returns the middle
   segment, not the last. Inferable: yes — ARN structure is documented AWS format.
7. `IsIAMNoSuchEntityException` unwraps (`errors.As`) to the typed exception; nil → false.
   Inferable: yes.
