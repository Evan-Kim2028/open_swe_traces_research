# API left after excision

```go
type Pattern interface {
    Match(path []string) bool
}
func ParsePattern(p string, domain []string) Pattern
```

`ParsePattern` still splits on `/` and stores `domain`. `Match` panics.
