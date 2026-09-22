# Bug report

The JSON transformer is broken: `Transform` doesn't descend into nested maps/slices, string
transforms aren't applied (or don't write back), object transforms run after children instead
of before, slice transforms can't replace elements, paths lack the dotted/`[]` structure,
unhandled types don't error, and `SortSlice` doesn't sort by JSON encoding.

Expected: recursive in-place walk; object-transforms-first ordering; `path[]` slice syntax;
string transforms composed in order; marshal-key sorting.

Reproduce with:

```
go test -count=1 ./pkg/jsonutils/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
