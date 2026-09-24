# Exported API — markdown

`MarkdownService.Render(ctx, text, opts)` POSTs `text` to `/markdown`
and returns rendered HTML. `opts` (`*MarkdownOptions`) may carry `Mode`
(markdown vs gfm) and `Context` (a repo path used to resolve
`@mentions` and issue references). Empty fields are omitted from the
request; `opts == nil` renders with defaults.
