# Exported API — memfs

`NewMemFSPath(context *MemFSContext, location string) *MemFSPath` — path under the context root (via Join).

`(*MemFSPath) Join(relativePath ...string) Path` — creates intermediate children.

`CreateFile` fails with `os.ErrExist` if contents already exist; `WriteFile` overwrites. `ReadFile` / `WriteTo` fail with `os.ErrNotExist` when empty. `ReadDir` is immediate children; `ReadTree` is the recursive set of leaf paths (nodes that have no children). `Remove` clears contents; `RemoveAll` removes every leaf under the node.

Callers: tests and in-memory VFS. In-tree tests: `TestMemFsCreateFile`, `TestMemFsReadDir`, `TestMemFsReadTree`.
