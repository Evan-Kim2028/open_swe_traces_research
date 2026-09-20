# Bug report

Collecting terraform objects to render is broken: resource and data-source lookups come back empty or wrongly grouped, names with dots, slashes, colons or leading digits are emitted raw instead of legalized, two different names that legalize to the same identifier are silently accepted, scalar and array outputs for the same key are merged instead of rejected, duplicate array elements are kept, and registering the same provider twice with different arguments is ignored.

Expected: resource names are legalized for terraform (`a.b` → `a-b`, `a/b` → `a--b`, `a:b` → `a_b`, leading digit → `prefix_` prefix); collisions on the legalized name are errors; a scalar-then-array output key is an error; output arrays are deduplicated and sorted.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/cloudup/terraformWriter/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
