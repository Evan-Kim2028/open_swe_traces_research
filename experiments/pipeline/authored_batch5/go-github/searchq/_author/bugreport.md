# Bug report — searchq

Search requests go out with no assembled Accept header: the per-type
preview media types and the text-match media type are all dropped, so
`opts.TextMatch` never produces match metadata and preview-gated search
kinds lose their required Accept values. `repository_id` is also never
sent.

Expected: Accept assembled from the search-type previews plus the
text-match media type when requested; `repository_id` forwarded as a
query param.

Got: bare request, no Accept assembly, no repository_id.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
