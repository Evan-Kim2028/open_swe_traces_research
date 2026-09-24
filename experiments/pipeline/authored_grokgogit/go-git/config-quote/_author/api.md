# API left after excision

```go
type Encoder struct
func NewEncoder(w io.Writer) *Encoder
func (e *Encoder) Encode(cfg *Config) error
```

`NewEncoder` is intact. `Encode`, `encodeSection`, `encodeSubsection`, and `encodeOptions` panic. The two `strings.Replacer` values exist but are empty.
