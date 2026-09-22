# Bug report

Calling any released HTTP file entry point panics. Expected: each
function returns the files already planned in `data`, after verifying the
`genpkg` argument equals `data.GenPkg()`; mismatches panic. Got: panics on
every call.

Reproduce with:

```
go test -count=1 ./http/codegen/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.
