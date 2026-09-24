# API left after excision

```go
func (a *AdvRefs) Encode(w io.Writer) error
```

`AdvRefs` still has `Version`, `Capabilities`, `References`, and `Shallows`. The file comment on `Encode` remains: payloads end with newline; capabilities, refs, and shallows are alphabetical except peeled refs follow their base.
