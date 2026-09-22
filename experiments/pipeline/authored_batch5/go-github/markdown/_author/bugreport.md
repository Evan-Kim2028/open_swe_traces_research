# Bug report — markdown

`MarkdownService.Render` ignores `MarkdownOptions`: `Mode` and `Context`
are never sent, so a caller asking for `mode: "gfm"` or a repo `Context`
to resolve `@mentions` gets default-rendered output.

Expected: non-empty option fields populate the render request.

Got: only `text` is sent; options are silently dropped.

Reproduce with:

```
go test -count=1 ./...
```

Do not use web search or any tool that accesses the internet; work only from the repository.
