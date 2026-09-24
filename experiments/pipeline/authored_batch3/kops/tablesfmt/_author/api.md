# Exported API — tablesfmt

Package `util/pkg/tables` (importable as `example.internal/clustkit/util/pkg/tables`).

- `type Table` + `AddColumn(name, getter func)` — register named columns; `Render(items, out, columnNames...)` — sort rows and emit an aligned table with a header.
- `func SortByFunction(len int, swap func(int,int), less func(int,int) bool)` — adapt closures to `sort.Interface`.

Production callers: `pkg/commands` `kops get` table output (`GetCluster` etc.).
