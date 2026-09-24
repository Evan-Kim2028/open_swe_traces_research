# Bug report — createcommit

`CreateCommit` ignores signing: commits created with a `Signer` in
options, or with a `Verification.Signature` already set, arrive at the
API with no signature. Passing `nil` options also panics.

Expected: a configured signer produces the commit's signature field, and
a pre-set `Verification.Signature` is sent as-is; `nil` options behave
like empty options.

Got: the `signature` field is never populated, and `opts == nil`
dereferences a nil pointer.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
