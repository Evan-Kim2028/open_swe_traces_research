# Details — tablesfmt

1. `AddColumn` stores the getter's reflect.Value keyed by name; a duplicate name overwrites. Inferable: yes.
2. `getFromValue` invokes the getter on the item and stringifies the FIRST return value via `reflectutils.ValueAsString`. Inferable: yes.
3. `SortByFunction`/`funcSorter` adapt len/less/swap closures onto sort.Interface — `sort.Sort` semantics. Inferable: yes.
4. `findColumns` resolves each name and errors "column not found: <name>" on the FIRST missing one. Inferable: partially.
5. `Render` kills the process (klog.Fatal) when `items` is not a slice — not an error return. Inferable: no — fatal-vs-error is arbitrary.
6. Rows sort lexicographically by column VALUES left-to-right (string compare, ties fall through to later columns). Inferable: partially.
7. Every emitted cell — header and data — is wrapped in `tabwriter.Escape` and tab-separated; the writer uses minwidth 0, tabwidth 8, padding 1, StripEscape. Inferable: no — the escape+tabwriter spellings are arbitrary.
