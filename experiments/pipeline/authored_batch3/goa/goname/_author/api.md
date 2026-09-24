# Exported API — goname

```go
func Comment(elems ...string) string
func Indent(s, prefix string) string
func SnakeCase(name string) string
func KebabCase(name string) string
func WrapText(text string, maxChars int) string
func Goify(str string, firstUpper bool) string
func GoifyAtt(att *expr.AttributeExpr, name string, upper bool) string
```

`runeSpacePos`, `runeSpacePosRev`, `fixReservedGo` are unexported
helpers.

## Pre-existing callers

Every generated Go file calls Goify/GoifyAtt for identifiers, Comment
for doc comments, SnakeCase/KebabCase for file and tag names, and
WrapText/Indent for comment layout.
