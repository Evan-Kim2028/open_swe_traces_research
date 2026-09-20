# Exported API — fidownload

Package `upup/pkg/fi` (importable as `example.internal/clustkit/upup/pkg/fi`).

- `func DownloadURL(ctx, url, dest string, hash *hashing.Hash) (*hashing.Hash, error)` — download to a path, skipping the fetch when the existing file already matches `hash`.
- `func OpenURL(url string) (io.ReadCloser, error)` — hardened HTTP GET stream (timeout, status check, cancel-on-close).
- `type cancelOnCloseReadCloser` — wraps a response body so `Close` also cancels the request context.

Production callers: `upup/pkg/fi/cloudup/apply_cluster.go` (asset downloads), nodeup asset installer.
