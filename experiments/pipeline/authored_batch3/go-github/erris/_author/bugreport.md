# Bug report — erris

Matching API errors with `errors.Is` stopped working: an error that should
equal an expected instance no longer matches, so code that branches on the
error kind or compares against a reference error misbehaves.

Expected: two error values with the same fields compare equal through
`errors.Is`, and differing ones do not.

Got: `errors.Is` reports no match even when the errors carry identical
information.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
