# Exported API — oamarshal

```go
func MarshalJSON(v any, extensions map[string]any) ([]byte, error)
func MarshalYAML(v any, extensions map[string]any) (any, error)
```

## Pre-existing callers

Every `MarshalJSON`/`MarshalYAML` method in `v2/openapi.go`, `v3/openapi.go`,
and `tags.go` delegates here, passing its `Extensions` map.
