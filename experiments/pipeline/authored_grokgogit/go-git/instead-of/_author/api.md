# API left after excision

```go
type URL struct {
    Name       string
    InsteadOfs []string
}
func (u *URL) Validate() error
func (u *URL) ApplyInsteadOf(url string) string
```

`Validate` still requires a non-empty `InsteadOfs`. `applyLongestInsteadOfMatch` panics. Config unmarshalling still fills `URLs`.
