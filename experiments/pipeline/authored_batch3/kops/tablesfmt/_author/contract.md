# Contract (L2) — tablesfmt

`AddColumn` registers a named reflect.Getter (dupes overwrite). `getFromValue` calls the getter and stringifies via `reflectutils.ValueAsString`. `SortByFunction` adapts len/less/swap closures to `sort.Interface`. `findColumns` resolves names, erroring on the first miss. `Render` fatals on non-slice input, sorts rows lexicographically by selected column values left-to-right, and emits a header + rows where every cell is `tabwriter.Escape`-wrapped, tab-separated, through a tabwriter (minwidth 0, tabwidth 8, padding 1, StripEscape).

## Coverage of original in-tree tests

No in-tree tests existed for this package (`util/pkg/tables` has no `*_test.go`). Reachability is via `pkg/commands` `kops get` table output.
