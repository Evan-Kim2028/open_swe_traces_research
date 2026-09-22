# API left after excision

```go
func NewEncoder(w io.Writer, h hash.Hash, opts ...Option) *Encoder
func (e *Encoder) Encode(idx *Index) error
```

Header, per-entry stat fields, v2/v3 raw names, and the footer hash still run. `encodeEntryNameV4`, `commonPrefixLen`, and `padEntry` panic. v2/v3 encode still calls `padEntry` after each name.
