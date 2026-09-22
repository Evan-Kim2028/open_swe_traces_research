# Exported API — firesources

Package `upup/pkg/fi` (importable as `example.internal/clustkit/upup/pkg/fi`).

`Resource` (`Open() (io.Reader, error)`) implementations and comparison helpers.

- `func ResourcesMatch(a, b Resource) (bool, error)` — streams both resources and compares
  chunked; equal only if every byte matches. Open/read errors propagate.
- `func CopyResource(dest io.Writer, r Resource) (int64, error)` — copies opened contents.
- `func ResourceAsString(r Resource)` / `ResourceAsBytes(r Resource)` — buffered copies.
- `NewStringResource(s)` — `Open` yields the string; `MarshalJSON` emits the string.
- `NewBytesResource(data)` — `Open` yields the bytes; `MarshalJSON` emits them AS A STRING.
- `NewFileResource(path)` — opens an os file; not-exist errors propagate unwrapped.
- `NewVFSResource(path vfs.Path)` — reads via `Path.ReadFile`; not-exist unwrapped.
- `TaskDependentResource[T]` — wraps a `Resource` produced by a `Task`; `Open` errors
  `resource opened before it is ready` until `Task` fills `Resource`; `IsReady` reports
  `Resource != nil`; `GetDependencies` returns the producing task.
- `FunctionToResource(fn)` — lazy resource; calls `fn` once on first `Open` and caches.

Example: `ResourcesMatch(NewStringResource("a"), NewStringResource("a"))` → true.
