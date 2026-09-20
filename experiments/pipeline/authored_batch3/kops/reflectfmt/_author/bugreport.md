# Bug report

Walking structs/maps/slices to visit every value is broken: visitors are not invoked on nested values, the documented skip signal does not prune subtrees, JSON field names are not honored when requested, and merging one struct's fields into another leaves the destination unchanged. Type names and formatted values come back wrong.

Expected: every exported nested field is visited depth-first; the skip sentinel prunes a subtree but not its siblings; with JSON names enabled a tagged field reports its JSON name; merging `{"b":1}` into a struct with a `B` field sets it.

Reproduce with:

```
go test -count=1 ./util/pkg/reflectutils/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
