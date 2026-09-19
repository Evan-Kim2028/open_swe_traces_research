# Contract (L2) — flagbuilder

Walking a config struct, only fields tagged `flag` become command-line flags. Empty tag still descends into nested structs. Tag `-` skips the subtree. Optional `,repeat` on a `[]string` emits one `--name=item` per element; otherwise the slice is a single comma-joined flag. Nil maps/slices/pointers (except `*string`) emit nothing.

`map[string]string` values become sorted `k=v` pairs comma-joined as one flag. `*string` with `flag-include-empty` always emits; otherwise empty and `flag-empty` matches are omitted. Bools and ints omit when their printed form equals `flag-empty`. metav1.Duration uses Go’s duration string, except `0` is rewritten `0s`. resource.Quantity uses the decimal representation. The flag list is sorted for stability.

The space-separated string form quotes a value if it contains a double quote (`%q`); the argv form never quotes.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestBuildKCMFlags` | durations, ints, nested structs, omit-empty |
| `TestKubeletConfigSpec` | kubelet tags including maps/slices |
| `TestBuildAPIServerFlags` | admission slices and repeat vs join |
| `TestBuildFlagsQuoting` | quote only when the joined-string form sees `"` |
