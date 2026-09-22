# Contract — vfsimpl

VFS path semantics in `util/pkg/vfs` (S3, GCS, Azure Blob, Kubernetes).
Every commitment below is covered by a hidden test; every hidden test maps
to a commitment.

## Commitments

1. **Path/String formats.** `scheme://bucket/key` for S3 (the scheme field
   is honoured, so non-`s3` schemes work), `gs://bucket/key`,
   `azureblob://account/container/key`, `k8s://host/key`. `String()`
   delegates to `Path()`. Covered by `TestDetail01`.
2. **Join.** Prepends the existing key to the relative parts and
   `path.Join`s — `.`/`..` are normalized — and the result keeps the
   bucket/scheme/host/context fields unchanged. Covered by `TestDetail02`.
3. **Base.** `Base()` is `path.Base(key)`: the last element, degenerate
   `.`/`/` for empty keys. Covered by `TestDetail03`.
4. **S3 HTTPS URL.** `S3Path.GetHTTPsUrl` resolves the bucket's region and
   builds a regional endpoint — not the fixed `s3.amazonaws.com` form —
   appending the key; dualstack mode selects a dualstack endpoint. (The
   region probe short-circuits to `S3_REGION` when `S3_ENDPOINT` is set, so
   no network is involved.) Covered by `TestDetail04`.
5. **GCS HTTPS URL.** `GSPath.GetHTTPsUrl` is the static
   `https://storage.googleapis.com/<bucket>/<key>` form with a trailing
   slash trimmed. Covered by `TestDetail05`.
6. **GCS not-found.** `isGCSNotFound` accepts the typed SDK sentinels
   (including wrapped) and `googleapi` 404s; other codes, generic errors,
   and nil are not not-found. Covered by `TestDetail06`.
7. **GSAcl.String (shape).** Renders the ACL entries inside braces, each
   via `%+v`; the exact brace/separator layout is not pinned. Covered by
   `TestDetail07`.
8. **TerraformLink (shape).** Emits a literal referencing
   `google_storage_bucket_object` and the given name; the attribute name is
   not pinned. Covered by `TestDetail08`.
9. **AWSErrorCode.** Unwraps (including through `fmt.Errorf` wrapping) a
   `smithy.APIError` and returns its code; non-API errors and nil yield "".
   Covered by `TestDetail09`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | yes — all four formats + String delegation |
| TestDetail02 | 2 | yes — normalization + field preservation |
| TestDetail03 | 3 | yes — path.Base semantics |
| TestDetail04 | 4 | partially — regional/dualstack resolution asserted; literal host not pinned |
| TestDetail05 | 5 | yes — static form + slash trim |
| TestDetail06 | 6 | partially — sentinels + 404 asserted; code list not pinned |
| TestDetail07 | 7 | no — brace shape + per-entry `%+v` only |
| TestDetail08 | 8 | no — resource type + name shape only |
| TestDetail09 | 9 | partially — unwrap/code asserted; non-API → "" |
