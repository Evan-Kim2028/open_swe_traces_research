# API left after excision

```go
func getStorageSize(v any) (int64, error)
```

Parses a config value that may be a raw `int64` or a size string with a unit suffix. Used for JetStream max-store / max-memory fields.
