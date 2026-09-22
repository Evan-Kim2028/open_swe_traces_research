# Bug report

Feature flags are broken: `ParseFlags` ignores `+`/`-` prefixes so flags can't be toggled,
unknown names crash or silently register, `Enabled()` ignores explicit settings or defaults,
re-registering a flag clobbers its default, and `Get` doesn't error on unknown names.

Expected: `KOPS_FEATURE_FLAGS="+A,-B"` enables A and disables B; bare names enable; unknown
names log-and-skip; explicit value > registered default > false; `new` returns the existing
flag for a repeated key.

Reproduce with:

```
go test -count=1 ./pkg/featureflag/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
