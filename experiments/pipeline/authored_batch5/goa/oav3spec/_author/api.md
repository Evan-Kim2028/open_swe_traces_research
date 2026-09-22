# Exported API — oav3spec

```go
func (i Info) MarshalJSON() ([]byte, error)
func (p PathItem) MarshalJSON() ([]byte, error)
func (o Operation) MarshalJSON() ([]byte, error)
func (p *Parameter) MarshalJSON() ([]byte, error)
func (r Response) MarshalJSON() ([]byte, error)
func (s SecurityScheme) MarshalJSON() ([]byte, error)
func (i Info) MarshalYAML() (any, error)
func (p PathItem) MarshalYAML() (any, error)
func (o Operation) MarshalYAML() (any, error)
func (p *Parameter) MarshalYAML() (any, error)
func (r Response) MarshalYAML() (any, error)
func (s SecurityScheme) MarshalYAML() (any, error)
func (s SecurityRequirements) IsZero() bool

// exampler interface (initExamples caller):
func (m *MediaType) setExample(val any)
func (m *MediaType) setExamples(val map[string]*ExampleRef)
func (h *Header) setExample(val any)
func (h *Header) setExamples(val map[string]*ExampleRef)
func (p *Parameter) setExample(val any)
func (p *Parameter) setExamples(val map[string]*ExampleRef)
```

The marshalers and `IsZero` are consumed implicitly by `encoding/json`
(`Marshaler`, `omitzero`) and `gopkg.in/yaml.v3` (`Marshaler`,
`omitempty`). The setters satisfy the unexported `exampler` interface.

## Pre-existing callers

`openapiv3.Files`/`openapiv3.New` build spec trees whose section
templates marshal them; `initExamples` (example.go) calls the setters on
`MediaType`, `Header`, and `Parameter` values while initializing
examples. `openapi.MarshalJSON`/`openapi.MarshalYAML` (marshal.go) are
the shared helpers gold delegates to; the `_Info`/`_PathItem`/
`_Operation`/`_Parameter`/`_Response`/`_SecurityScheme` alias types exist
to break marshal recursion.
