# Closure — vfspaths

Package: `util/pkg/vfs` (`example.internal/clustkit/util/pkg/vfs`).

Files: `util/pkg/vfs/context.go` (13 funcs).

Removed functions (bodies stubbed): `VFSContext.BuildVfsPath`, `nextBackoffDuration`, `RetryWithBackoff`, `buildS3Path`, `buildDOPath`, `buildLinodePath`, `buildHetznerPath`, `buildKubernetesPath`, `buildMemFSPath`, `buildGCSPath`, `buildOpenstackSwiftPath`, `buildAzureBlobPath`, `buildSCWPath`.

Exported entry point(s): `(*VFSContext).BuildVfsPath` and `RetryWithBackoff` — the URL→Path factory and the retry loop used by `ReadFile`/`readHTTPLocation`.
