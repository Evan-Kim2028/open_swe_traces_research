# Closure — vfsimpl

Package: `util/pkg/vfs` (`example.internal/clustkit/util/pkg/vfs`).

Files: `s3fs.go` (8), `gsfs.go` (10), `azureblob.go` (7), `k8sfs.go` (6) — 31 funcs.

Removed functions (bodies stubbed): S3 `Path`/`Bucket`/`Key`/`String`/`Join`/`Base`/
`GetHTTPsUrl` + `AWSErrorCode`; `GSAcl.String`, GS `Path`/`Bucket`/`Object`/`String`/`Join`/
`Base`/`GetHTTPsUrl`/`isGCSNotFound`/`TerraformLink`; Azure `Account`/`Container`/`Key`/
`Base`/`Path`/`String`/`Join`; K8s `Path`/`Host`/`Key`/`String`/`Join`/`Base`.

Exported entry point(s): `Path.Join`/`Base`/`String` are used by every state-store path
manipulation; `GetHTTPsUrl` feeds `kops get` browsing and asset URLs.

Test files removed in excision: `s3fs_test.go`, `gsfs_test.go`, `azureblob_test.go`,
`s3context_test.go`.
