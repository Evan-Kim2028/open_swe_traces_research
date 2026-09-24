# Bug report — propvalues

The typed accessors for custom-property default values always report
failure: even when a property's type and payload clearly match the accessor,
it refuses to hand back the value.

Expected: each accessor returns the payload in its Go type when the
property's value type calls for it, and a not-ok result only when it does
not.

Got: every accessor returns a not-ok result in all cases.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
