# Bug report — pagevalues

Cursor-paginated endpoints no longer paginate: iterating a list whose responses
advertise a cursor link stops after the first page instead of following the
cursor. Links that carry a `since` parameter instead of `page` are also
ignored, so endpoints using that style report no next page.

Expected: all advertised pagination links are honored — cursor links, plain
page links, and since-style links — so iterators and `NextPage`-style loops
walk every page.

Got: cursor and since links are dropped; only `page`/`before`/`after` links
work.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
