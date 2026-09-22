# Bug report

`pkg/kubemanifest` is broken: `LoadObjectsFrom` errors on empty/comment-only YAML sections and
loses multi-doc boundaries, `ObjectList.ToYAML` emits empty objects or wrong separators, the
`Kind`/`GetName`/`GetNamespace`/`APIVersion` getters panic or return wrong values on missing
fields, `Reparse`/`Set` don't walk nested map paths, `Set` writes the raw value instead of a
remarshalable map, and the `visit` walker doesn't recurse into maps/slices or apply mutators.

Expected: comment-only sections skipped; `""` for absent/wrong-typed fields; nested paths
navigated as `map[string]interface{}` with errors naming the field; in-place mutation via
mutators; `[]string` skipped by the walker.

Reproduce with:

```
go test -count=1 ./pkg/kubemanifest/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
