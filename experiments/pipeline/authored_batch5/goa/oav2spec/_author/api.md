# Exported API — oav2spec

```go
func (i Info) MarshalJSON() ([]byte, error)
func (p Path) MarshalJSON() ([]byte, error)
func (o Operation) MarshalJSON() ([]byte, error)
func (p Parameter) MarshalJSON() ([]byte, error)
func (r Response) MarshalJSON() ([]byte, error)
func (s SecurityDefinition) MarshalJSON() ([]byte, error)
func (i Info) MarshalYAML() (any, error)
func (p Path) MarshalYAML() (any, error)
func (o Operation) MarshalYAML() (any, error)
func (p Parameter) MarshalYAML() (any, error)
func (r Response) MarshalYAML() (any, error)
func (s SecurityDefinition) MarshalYAML() (any, error)
func (s SecurityRequirements) IsZero() bool
```

All thirteen satisfy `json.Marshaler` / `yaml.Marshaler` / the
`omitzero` contract consumed implicitly by `encoding/json` and
`gopkg.in/yaml.v3` — no call site names them directly.

## Pre-existing callers

`openapiv2.Files` produces `codegen.File` values whose section templates
marshal the spec tree; every swagger fixture test renders through these
methods. `openapi.MarshalJSON`/`openapi.MarshalYAML` (in
`http/codegen/openapi/marshal.go`) are the pre-existing helpers the gold
bodies delegate to; the `_Info`/`_Path`/`_Operation`/`_Parameter`/
`_Response`/`_SecurityDefinition` alias types exist solely to break
marshal recursion.
