# Details — vfsimpl

1. `Path()` formats: `scheme://bucket/key` (S3 — scheme field, so `s3` or custom),
   `gs://bucket/key`, `azureblob://account/container/key`, `k8s://host/key`. `String()`
   delegates to `Path()`. Inferable: yes — schemes match the URL builders elsewhere.
2. `Join` prepends the existing key to the relative parts and `path.Join`s — so `.`/`..`
   are normalized, and the result keeps bucket/host/context fields unchanged. Inferable: yes.
3. `Base` is `path.Base(key)` — the last element, or `.`/`/` for degenerate keys.
   Inferable: yes.
4. `S3Path.GetHTTPsUrl` does a bucket-region lookup then resolves the regional S3 endpoint
   (dualstack-aware) and appends the key — NOT a fixed `s3.amazonaws.com` format.
   Inferable: partially — resolver-based construction is a choice.
5. `GSPath.GetHTTPsUrl` is the static `storage.googleapis.com/bucket/key` form with a
   trailing slash trimmed. Inferable: yes.
6. `isGCSNotFound` checks typed sentinel errors AND googleapi 404 — an SDK sentinel wins
   first. Inferable: partially.
7. `GSAcl.String` renders `{a, b}` using `%+v` of each entry. Inferable: no — brace layout
   arbitrary.
8. `TerraformLink` emits a `google_storage_bucket_object` `output_name` property literal.
   Inferable: no — attribute name arbitrary.
9. `AWSErrorCode` (vfs-local duplicate of the awsup helper) unwraps `smithy.APIError`.
   Inferable: partially.
