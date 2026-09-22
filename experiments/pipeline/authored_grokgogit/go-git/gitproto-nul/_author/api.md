# API left after excision

```go
var ErrInvalidGitProtoRequest error
type GitProtoRequest struct {
    RequestCommand string
    Pathname       string
    Host           string
    ExtraParams    []string
}
func (g *GitProtoRequest) Encode(w io.Writer) error
func (g *GitProtoRequest) Decode(r io.Reader) error
```

`validate` and `validateGitProtoField` panic. Comments on the type still describe the NUL-framed pkt-line.
