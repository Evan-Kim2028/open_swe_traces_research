# Contract (L2) — htmlrender

Instance(name, data) packages an HTML render. The production renderer carries an already-compiled template and reuses it for every render. The debug renderer re-parses templates on every render: it uses Files when non-empty, else the Glob pattern, else the FileSystem plus Patterns through an fs.FS adapter — and panics when none is configured. A nil FuncMap is treated as empty and custom Delims are applied to the parsed template. Render writes the HTML content type first; a nil template yields the 'html renderer is not configured' error; with an empty Name it executes the template's own root, otherwise it executes the named template; execution errors propagate.

## Coverage of original in-tree tests

| original test | contract sentence |
|---|---|
| `TestRenderHTMLTemplate` | production instance executes the named template |
| `TestRenderHTMLWithoutTemplate` | nil template returns the not-configured error |
| `TestRenderHTMLTemplateEmptyName` | empty Name executes the template root |
| `TestRenderHTMLDebugFiles/Glob/FS` | debug renderer re-parses from files, glob, or FS+patterns per render |
| `TestRenderHTMLDebugPanics` | debug renderer with no source configured panics |
| `TestRenderHTMLTemplateError/ExecuteError` | parse/execute failures propagate |



Reproduce with:

```
tests/test.sh
```

That script installs the hidden suite and runs it (equivalent to
`go test -count=1 -timeout 15m ./render/`
after the suite is in the tree). Do not skip, delete, or weaken the
tests. Do not change test assertions or testdata just to make them
green.

Work in `/app`. Keep unrelated tests passing.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
