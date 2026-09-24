# Bug report — gitrefs

Ref mutations lost their input handling: `CreateRef` accepts empty
`Ref`/`SHA` and sends the ref without the `refs/` prefix Git expects;
`UpdateRef` accepts empty `ref`/`SHA`; and both `UpdateRef`/`DeleteRef`
send `refs/heads/x` verbatim into the route — double-prefixing it —
instead of trimming and escaping it.

Expected: required fields validated, `refs/` normalized on create and
stripped on update/delete, and the ref path-escaped.

Got: no validation, no normalization, malformed routes for prefixed
refs.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
