# API left after excision

```go
func (c *client) writeLeafSub(w *bytes.Buffer, key string, n int32)
```

Appends one leaf-node interest protocol line for `key` with interest count `n`. Caller holds the client lock. `CR_LF` and `digits` remain in the package.
