# API left after excision

Unexported:

```go
func foldKey(name string) string
func foldRune(r rune) rune
```

Callers in the decoder still do `foldKey(sectionName)` as a map key so that `strings.EqualFold` names collide. The long comment above `foldKey` is still in the file and states the EqualFold contract.
