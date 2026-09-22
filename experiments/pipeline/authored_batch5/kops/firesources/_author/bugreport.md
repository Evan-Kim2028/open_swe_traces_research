# Bug report

The `fi` resource helpers are broken: `ResourcesMatch` reports equal for different contents (or
errors on EOF instead of comparing), `CopyResource` wraps not-found errors so callers can no
longer detect missing files, `ResourceAsString`/`ResourceAsBytes` truncate, `MarshalJSON` on the
byte/string resources emits the wrong shape, `FileResource`/`VFSResource` open failures are
misclassified, `TaskDependentResource` opens before its task ran, and `FunctionToResource`
re-invokes its function on every `Open` instead of caching.

Expected: byte-exact streamed comparison; unwrapped `os.IsNotExist`; lazy+memoized function
resources; readiness gating on `TaskDependentResource`.

Reproduce with:

```
go test -count=1 ./upup/pkg/fi/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the
internet; work only from the repository.
