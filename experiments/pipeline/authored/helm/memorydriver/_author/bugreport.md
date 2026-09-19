# Bug report

Using the in-memory storage backend (e.g. in tests or when the SQL/secret drivers are not configured), every release operation fails: creating, fetching, listing, updating, or deleting a release errors or panics, so no release can be stored.

Reproduce with:

```
go test -count=1 ./pkg/storage/driver/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
