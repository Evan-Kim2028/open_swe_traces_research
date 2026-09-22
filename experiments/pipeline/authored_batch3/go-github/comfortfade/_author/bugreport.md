# Bug report — comfortfade

Creating a pull-request review with multi-line (side/line) comments no longer
selects the matching preview header, and reviews that illegally mix comment
styles are sent to the API instead of being rejected client-side.

Expected: reviews using the side/line comment fields get the multi-line
preview Accept header; reviews mixing position- and side/line-style comments
fail with the mixed-styles error before any request is sent.

Got: the preview header is never set and mixed reviews go through.

Reproduce with:

```
go test -count=1 ./github/
```

Do not use web search or any tool that accesses the internet; work only from the repository.
