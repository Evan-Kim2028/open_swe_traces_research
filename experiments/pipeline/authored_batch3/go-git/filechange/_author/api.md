# Exported API — filechange

Package `plumbing/object` (module `example.internal/gitkit/v6`) — tree-file accessors and
change-pair predicates.

`File{Name, Mode, Blob}`: `Contents() (string, error)`, `IsBinary() (bool, error)`,
`Lines() ([]string, error)`. `Change{From, To ChangeEntry}`: `Action()
(merkletrie.Action, error)`, `Files() (*File, *File, error)`, `String`. `Changes`
sortable slice with `Less`, `String`, `Patch`.

Callers: difftree output, patch generation, blame/status paths. In-tree tests removed: 2.
