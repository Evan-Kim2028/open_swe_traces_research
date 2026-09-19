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



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./pkg/flagbuilder/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
