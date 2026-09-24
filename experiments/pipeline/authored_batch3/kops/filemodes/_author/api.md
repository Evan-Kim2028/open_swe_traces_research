# Exported API — filemodes

Package `upup/pkg/fi` (importable as `example.internal/clustkit/upup/pkg/fi`).

- `func ParseFileMode(s string, defaultMode os.FileMode) (os.FileMode, error)` — parse an octal mode string.
- `func FileModeToString(mode os.FileMode) string` — render a mode as an octal string.
- `func EnsureFileMode(destPath string, fileMode os.FileMode) (changed bool, err error)` — chmod only on drift.
- `func fileHasHash(f string, expected *hashing.Hash) (bool, error)` — compare a file's content hash (unexported, used by `DownloadURL`).

Production callers: `WriteFile` (same file), `DownloadURL` in `http.go`, `upup/pkg/fi/nodeup` command paths.
