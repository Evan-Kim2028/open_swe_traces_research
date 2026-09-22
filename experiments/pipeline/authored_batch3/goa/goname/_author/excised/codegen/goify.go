package codegen

import (
	"example.internal/apikit/v3/expr"
	_ "example.internal/apikit/v3/internal/codegenname"
)

// Goify makes a valid Go identifier out of any string. It does that by removing
// any non letter and non digit character and by making sure the first character
// is a letter or "_". Goify produces a "CamelCase" version of the string, if
// firstUpper is true the first character of the identifier is uppercase
// otherwise it's lowercase.
func Goify(str string, firstUpper bool) string {
	panic("excised: Goify")
}

// GoifyAtt honors any struct:field:name meta set on the attribute and calls
// Goify with the tag value if present or the given name otherwise.
func GoifyAtt(att *expr.AttributeExpr, name string, upper bool) string {
	panic("excised: GoifyAtt")
}

// fixReservedGo appends an underscore on to Go reserved keywords.
func fixReservedGo(w string) string {
	panic("excised: fixReservedGo")
}
