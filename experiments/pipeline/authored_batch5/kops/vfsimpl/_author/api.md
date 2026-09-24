# Exported API — vfsimpl

Package `util/pkg/vfs` (importable as `example.internal/clustkit/util/pkg/vfs`).

Path accessors for the S3/GCS/AzureBlob/Kubernetes `Path` implementations.

- `S3Path.Path()` — `scheme://bucket/key`; `Bucket`, `Key`, `String` (=Path);
  `Join(rel...)` — `path.Join` over key, preserving context/scheme/sse/optFn;
  `Base()` — `path.Base(key)`; `GetHTTPsUrl(dualstack)` — resolves the bucket's regional
  S3 endpoint and appends the key; `AWSErrorCode(err)` — `smithy.APIError` code or `""`.
- `GSAcl.String()` — `{acl1, acl2, ...}` via `%+v` per entry.
- `GSPath.Path()` — `gs://bucket/key`; `Bucket`, `Object` (=key), `String`, `Join`, `Base`;
  `GetHTTPsUrl()` — `https://storage.googleapis.com/<bucket>/<key>` minus trailing slash;
  `isGCSNotFound(err)` — `ErrObjectNotExist`/`ErrBucketNotExist` or googleapi 404;
  `TerraformLink(name)` — literal `google_storage_bucket_object.<name>.output_name`.
- `AzureBlobPath` — `Account`, `Container`, `Key`, `Base`, `Path` =
  `azureblob://account/container/key`, `String`, `Join`.
- `KubernetesPath` — `Path` = `k8s://host/key`; `Host`, `Key`, `String`, `Join`, `Base`.

Example: `S3Path{s3://b/a/c}.Join("d")` → `s3://b/a/c/d`; `Base()` → `d`.
