# Bug report

Merging `--set`-style key/value strings into a map is broken: dotted names no longer nest, indexed assignments like `l[0]=x` are rejected or lose entries, `{a,b}` brace lists do not produce lists, `true`/`false`/`null` scalars come back as plain strings instead of typed values, and backslash-escaped characters are not unescaped.

Expected: `a=b,c.d=e` produces `{"a":"b","c":{"d":"e"}}`; `l[0]=x,l[1]=y` produces `{"l":["x","y"]}`; `t=true` produces a bool and `n=null` produces a nil value; the always-string variant keeps every value a string.

Reproduce with:

```
go test -count=1 ./third_party/forked/helmstrvals/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
