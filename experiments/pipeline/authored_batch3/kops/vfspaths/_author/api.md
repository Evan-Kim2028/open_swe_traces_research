# Exported API — vfspaths

Package `util/pkg/vfs` (importable as `example.internal/clustkit/util/pkg/vfs`).

- `func (c *VFSContext) BuildVfsPath(p string) (Path, error)` — URL→`Path` dispatch: bare/`file://`→FSPath; `s3://`, `do://`, `linode://`, `hos://`, `scw://`→S3Path variants; `memfs://`, `gs://`, `k8s://`, `swift://`, `azureblob://`→their Path types.
- `func RetryWithBackoff(backoff wait.Backoff, condition func() (bool, error)) (bool, error)` — retry loop that returns the condition's own result on exhaustion.
- `var Context` / `NewVFSContext` / `NewTestingVFSContext`.

Production callers: `ReadFile`/`readHTTPLocation` (same file), `upup/pkg/fi/http.go` (`WriteToWithContext` dispatch), VFS-backed stores.
