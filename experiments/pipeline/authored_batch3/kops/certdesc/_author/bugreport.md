# Bug report

Certificate descriptions are broken: subject names render with numeric OID components instead of readable attribute names, key usages come out as numbers or in a different order each run, extended key usages fall back to raw integers, and a certificate's type no longer collapses to its short name — a CA cert describes itself as a list of usages instead of `ca`.

Expected: a subject renders comma-joined `name=value` pairs with readable attribute names; usages render as their flag names in a stable order; a certificate whose usages match a well-known combination reports the short type name.

Reproduce with:

```
go test -count=1 ./pkg/pki/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
