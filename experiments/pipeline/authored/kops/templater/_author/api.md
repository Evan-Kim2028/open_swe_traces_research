# Exported API — templater

`NewTemplater(channel *kops.Channel) *Templater`

`(*Templater) Render(content string, context map[string]interface{}, snippets map[string]string, failOnMissing bool) (string, error)`

`failOnMissing` sets `missingkey=error`. Snippets are parsed as additional templates; a snippet named `mainTemplate` is rejected. Funcs include sprig, `indent`, `include`, and channel helpers: recommended kubernetes upgrade version, recommended kops kubernetes version, recommended image. Indent does not pad the first line or empty lines. Include panics (caught by Render as error) if the snippet is missing.

Callers: addon/channel manifest rendering. In-tree tests: eight Render cases (ok, missing, indent, channel funcs, snippet, context, allow-missing, integration).
