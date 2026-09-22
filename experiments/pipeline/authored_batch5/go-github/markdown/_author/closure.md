# Closure — markdown

Package: github (root). File: github/markdown.go.
Removed bodies: MarkdownService.Render — stubbed to send only `Text`,
dropping the `opts != nil` block that copies non-empty `opts.Mode` and
`opts.Context` into the request. Rendering still works; the `markdown`
mode (gfm/plain) and repo context for `@mentions`/issue links are
silently dropped.
Kept: MarkdownOptions, markdownRenderRequest, client plumbing.
Tests removed: github/markdown_test.go deleted outright (it held a
single closure test, TestMarkdownService_Markdown).
