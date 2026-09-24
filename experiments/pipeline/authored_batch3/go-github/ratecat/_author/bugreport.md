# Bug report — ratecat

Every endpoint now lands in the same rate-limit bucket: search, graphql,
audit-log and other specially-metered calls all report against the core
pool, so rate-limit bookkeeping and wait-until-reset behavior are wrong for
non-core endpoints.

Expected: each endpoint is bucketed by its path (and sometimes method) into
the rate-limit pool GitHub actually meters.

Got: all endpoints classify into the core bucket.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
