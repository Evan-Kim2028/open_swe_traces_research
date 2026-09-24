// This file generates functions that check service values and values sent over
// HTTP, gRPC, and JSON-RPC. Each function uses the Go names already chosen for
// its package.
package codegen

import (
	"bytes"
	"fmt"
	_ "strconv"
	_ "strings"
	"text/template"

	"example.internal/apikit/v3/expr"
)

type (
	// nilUserTypeValidationAttributor reports generated user types whose
	// validator defines what nil means. Other user-type validators are called
	// only for a present value.
	nilUserTypeValidationAttributor interface {
		ValidationAcceptsNil(*expr.AttributeExpr) bool
	}

	// unionValidationCase describes one possible union branch in generated
	// validation code.
	unionValidationCase struct {
		// Type is the generated Go type for the branch.
		Type string
		// Field is the field which stores the branch value.
		Field string
		// Name is the branch name shown in validation errors.
		Name string
		// PayloadRequiresPresence is true when selecting this branch also
		// requires a non-nil value.
		PayloadRequiresPresence bool
		// Validation checks the value stored by this branch.
		Validation string
	}

	// unionValidationData contains the information needed to write one union
	// check.
	unionValidationData struct {
		// Target is the generated union value being checked.
		Target string
		// Context identifies the union in validation errors.
		Context validationPath
		// Protobuf is true when each selected branch is stored in its own generated
		// protobuf struct.
		Protobuf bool
		// Cases lists every branch accepted by the union.
		Cases []unionValidationCase
		// Apikit is the generated import name of Apikit's error package.
		Goa string
	}

	// validationPath stores an error path while Apikit writes validation source.
	// variable is true when root names a parameter in the generated function.
	validationPath struct {
		root     string
		suffix   string
		variable bool
	}
)

var (
	enumValT       *template.Template
	formatValT     *template.Template
	patternValT    *template.Template
	exclMinMaxValT *template.Template
	minMaxValT     *template.Template
	lengthValT     *template.Template
	requiredValT   *template.Template
	arrayValT      *template.Template
	mapValT        *template.Template
	unionValT      *template.Template
	unionSumValT   *template.Template
	userValT       *template.Template
)

func init() {
	fm := template.FuncMap{
		"slice":          toSlice,
		"oneof":          oneof,
		"constant":       constant,
		"validationPath": renderValidationPath,
		"isUnion": func(att *expr.AttributeExpr) bool {
			if att == nil {
				return false
			}
			return expr.IsUnion(att.Type)
		},
		"isSumType": func(scope Attributor) bool {
			if scope == nil {
				return false
			}
			return scope.IsSumType()
		},
		"isUnionPointer": func(ctx *AttributeContext, required bool) bool {
			return ctx.IsUnionPointer(required)
		},
		"add": func(a, b int) int { return a + b },
	}
	enumValT = template.Must(template.New("enum").Funcs(fm).Parse(codegenTemplates.Read(validationEnumT)))
	formatValT = template.Must(template.New("format").Funcs(fm).Parse(codegenTemplates.Read(validationFormatT)))
	patternValT = template.Must(template.New("pattern").Funcs(fm).Parse(codegenTemplates.Read(validationPatternT)))
	exclMinMaxValT = template.Must(template.New("exclMinMax").Funcs(fm).Parse(codegenTemplates.Read(validationExclMinMaxT)))
	minMaxValT = template.Must(template.New("minMax").Funcs(fm).Parse(codegenTemplates.Read(validationMinMaxT)))
	lengthValT = template.Must(template.New("length").Funcs(fm).Parse(codegenTemplates.Read(validationLengthT)))
	requiredValT = template.Must(template.New("req").Funcs(fm).Parse(codegenTemplates.Read(validationRequiredT)))
	arrayValT = template.Must(template.New("array").Funcs(fm).Parse(codegenTemplates.Read(validationArrayT)))
	mapValT = template.Must(template.New("map").Funcs(fm).Parse(codegenTemplates.Read(validationMapT)))
	unionValT = template.Must(template.New("union").Funcs(fm).Parse(codegenTemplates.Read(validationUnionT)))
	unionSumValT = template.Must(template.New("union-sum").Funcs(fm).Parse(codegenTemplates.Read(validationUnionSumT)))
	userValT = template.Must(template.New("user").Funcs(fm).Parse(codegenTemplates.Read(validationUserT)))
}

// AttributeValidationCode produces Go code that runs the validations defined
// in the given attribute against the value held by the variable named target.
//
// See ValidationCode for a description of the arguments.
func AttributeValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias bool, target, attName string) string {
	return recurseValidationCode(att, put, attCtx, req, alias, false, target, literalValidationPath(attName), nil).String()
}

// ValidationCode produces Go code that runs the validations defined in the
// given attribute and its children recursively against the value held by the
// variable named target.
//
// put is the parent UserType if any. It is used to compute proto oneof type names.
//
// attCtx is the attribute context used to generate attribute name and reference
// in the validation code.
//
// req indicates whether the attribute is required (true) or optional (false)
//
// alias indicates whether the attribute is an alias user type attribute.
//
// view indicates whether the attribute is a view type attribute.
// This only matters for union types: generated Apikit view union types have a
// different layout than proto generated union types.
//
// target is the variable name against which the validation code is generated
//
// context is used to produce helpful messages in case of error.
func ValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias, view bool, target string) string {
	return recurseValidationCode(att, put, attCtx, req, alias, view, target, literalValidationPath(target), nil).String()
}

// ValidationCodeWithPathParameter produces validation code whose error paths
// begin with the string held by pathParameter. target and pathParameter are Go
// expressions.
func ValidationCodeWithPathParameter(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias, view bool, target, pathParameter string) string {
	return recurseValidationCode(att, put, attCtx, req, alias, view, target, parameterValidationPath(pathParameter), nil).String()
}

func recurseValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias, view bool, target string, context validationPath, seen map[expr.UserType]*bytes.Buffer) *bytes.Buffer {
	return renderValidationCode(att, put, attCtx, req, alias, view, target, context, seen, true)
}

// renderValidationCode writes one validation tree. localGuards reports whether
// local rule templates must check a pointer before reading it. Nested fields
// disable those checks when validateAttribute wraps the whole field once.
func renderValidationCode(att *expr.AttributeExpr, put expr.UserType, attCtx *AttributeContext, req, alias, view bool, target string, context validationPath, seen map[expr.UserType]*bytes.Buffer, localGuards bool) *bytes.Buffer {
	if seen == nil {
		seen = make(map[expr.UserType]*bytes.Buffer)
	}
	var (
		buf      = new(bytes.Buffer)
		first    = true
		ut, isUT = att.Type.(expr.UserType)
	)

	// Break infinite recursions
	// Note: when alias=true, we're validating the underlying base type,
	// so alias types shouldn't use the recursion guard. Only non-alias user
	// types need cycle protection.
	if isUT && !alias {
		origin := ut.Origin()
		if buf, ok := seen[origin]; ok {
			return buf
		}
		seen[origin] = buf
	}

	newline := func() {
		if !first {
			buf.WriteByte('\n')
		} else {
			first = false
		}
	}

	// Write validations on attribute if any.
	validation := validationCode(att, attCtx, req, alias, target, context, localGuards)
	if validation != "" {
		buf.WriteString(validation)
		first = false
	}

	// Recurse down depending on attribute type.
	switch {
	case expr.IsObject(att.Type):
		if isUT {
			put = ut
		}
		for _, nat := range *(expr.AsObject(att.Type)) {
			tgt := fmt.Sprintf("%s.%s", target, attCtx.Scope.Field(nat.Attribute, nat.Name, true))
			ctx := context.child("." + nat.Name)
			val := validateAttribute(attCtx, nat.Attribute, put, tgt, ctx, att.IsRequired(nat.Name), view, seen)
			if val != "" {
				newline()
				buf.WriteString(val)
			}
		}
	case expr.IsArray(att.Type):
		arr := expr.AsArray(att.Type)
		elem := arr.ElemType
		ctx := attCtx
		if expr.IsPrimitive(elem.Type) {
			ctx = attCtx.Dup()
			ctx.Pointer = attCtx.IsArrayElementPointer(arr)
		}
		val := validateAttribute(ctx, elem, put, "e", context.child("[*]"), true, view, seen)
		nonNullableElems := arr.NonNullableElems &&
			(IsNilable(elem.Type) || attCtx.IsArrayElementPointer(arr))
		if val != "" || nonNullableElems {
			newline()
			data := map[string]any{
				"target":           target,
				"validation":       val,
				"checkNilElements": nonNullableElems,
				"context":          context,
				"goa":              "goa",
			}
			if err := arrayValT.Execute(buf, data); err != nil {
				panic(err) // bug
			}
		}
	case expr.IsMap(att.Type):
		m := expr.AsMap(att.Type)
		ctx := attCtx.Dup()
		ctx.Pointer = false
		keyVal := validateAttribute(ctx, m.KeyType, put, "k", context.child(".key"), true, view, seen)
		if keyVal != "" {
			keyVal = "\n" + keyVal
		}
		valueVal := validateAttribute(ctx, m.ElemType, put, "v", context.child("[key]"), true, view, seen)
		if valueVal != "" {
			valueVal = "\n" + valueVal
		}
		if keyVal != "" || valueVal != "" {
			newline()
			data := map[string]any{"target": target, "keyValidation": keyVal, "valueValidation": valueVal}
			if err := mapValT.Execute(buf, data); err != nil {
				panic(err) // bug
			}
		}
	case expr.IsUnion(att.Type):
		u := expr.AsUnion(att.Type)
		if attCtx.Scope.IsSumType() {
			cases := make([]map[string]any, 0, len(u.Values))
			for _, v := range u.Values {
				// Sum-type unions (struct-based, with Kind/AsX accessors) store each
				// branch as either a value (primitives, arrays, maps) or a pointer
				// (object user types). Request-body validation may already use value
				// semantics for nested objects, so preserve the enclosing context and
				// only keep pointer semantics when both layers use pointers.
				unionCtx := attCtx.Dup()
				unionCtx.Pointer = unionCtx.Pointer && expr.IsObject(v.Attribute.Type)
				val := validateAttribute(unionCtx, v.Attribute, put, "actual", context.child(".value"), true, view, seen)
				if val == "" {
					continue
				}
				cases = append(cases, map[string]any{
					"typeTag":    v.Name,
					"fieldName":  Goify(v.Name, true),
					"validation": val,
				})
			}
			if len(cases) > 0 {
				newline()
				data := map[string]any{
					"target": target,
					"cases":  cases,
					"goa":    "goa",
				}
				if err := unionSumValT.Execute(buf, data); err != nil {
					panic(err) // bug
				}
			}
			break
		}

		// Validate unions represented as interfaces (e.g., protobuf oneof wrappers).
		var cases []unionValidationCase
		for _, v := range u.Values {
			vatt := v.Attribute
			if view {
				// Union values in views are never pointers - they are concrete typed values
				unionCtx := attCtx.Dup()
				unionCtx.Pointer = false
				val := validateAttribute(unionCtx, vatt, put, "v", context.child(".value"), true, view, seen)
				if val != "" {
					cases = append(cases, unionValidationCase{
						Type:       attCtx.Scope.Ref(vatt, attCtx.Pkg(vatt)),
						Validation: val,
					})
				}
			} else {
				fieldName := attCtx.Scope.Field(vatt, v.Name, true)
				branchCtx := attCtx
				if expr.IsPrimitive(vatt.Type) {
					// A union wrapper stores its scalar directly. The wrapper
					// itself records whether that branch was selected.
					branchCtx = attCtx.Dup()
					branchCtx.Pointer = false
				}
				val := validateAttribute(branchCtx, vatt, put, "v."+fieldName, context.child(".value"), true, view, seen)
				parent := &expr.AttributeExpr{Type: put}
				tref := attCtx.Scope.Ref(parent, attCtx.Pkg(parent))
				cases = append(cases, unionValidationCase{
					Type:                    tref + "_" + fieldName,
					Field:                   fieldName,
					Name:                    v.Name,
					PayloadRequiresPresence: protobufUnionPayloadRequiresPresence(vatt),
					Validation:              val,
				})
			}
		}
		if len(cases) > 0 {
			newline()
			data := unionValidationData{
				Target:   target,
				Context:  context,
				Protobuf: !view,
				Cases:    cases,
				Goa:      "goa",
			}
			if err := unionValT.Execute(buf, data); err != nil {
				panic(err) // bug
			}
		}
	}

	return buf
}

// protobufUnionPayloadRequiresPresence reports whether selecting a protobuf
// union branch requires a non-nil value. Messages, byte slices, and Any values
// may be nil in Go, so their generated checks must reject nil explicitly.
func protobufUnionPayloadRequiresPresence(att *expr.AttributeExpr) bool {
	panic("excised: protobufUnionPayloadRequiresPresence")
}

func validateAttribute(ctx *AttributeContext, att *expr.AttributeExpr, put expr.UserType, target string, context validationPath, req, view bool, seen map[expr.UserType]*bytes.Buffer) string {
	panic("excised: validateAttribute")
}

// validationAttributeNeedsNilGuard reports whether a nested value may be nil
// in the generated Go layout and must be checked before any validation uses it.
func validationAttributeNeedsNilGuard(att *expr.AttributeExpr, ctx *AttributeContext, required bool) bool {
	panic("excised: validationAttributeNeedsNilGuard")
}

// validationCode produces Go code that runs the validations that effectively
// apply to the given attribute - see expr.EffectiveValidation - if any
// against the content of the variable named target. The generated code
// assumes that there is a pre-existing "err" variable of type error. It
// initializes that variable in case a validation fails. validationCode is
// pure: it never mutates att or any expression reachable from it.
//
// attCtx is the attribute context
//
// req indicates whether the attribute is required (true) or optional (false)
//
// alias indicates whether the attribute is an alias user type attribute.
//
// view indicates whether the attribute is a view type attribute.
// This only matters for union types: generated Apikit view union types have a
// different layout than proto generated union types.
//
// target is the variable name against which the validation code is generated
//
// context is used to produce helpful messages in case of error.
func validationCode(att *expr.AttributeExpr, attCtx *AttributeContext, req, alias bool, target string, context validationPath, localGuards bool) string {
	panic("excised: validationCode")
}

// literalValidationPath folds a complete error path into a quoted Go string
// while Apikit is generating source.
func literalValidationPath(root string) validationPath {
	panic("excised: literalValidationPath")
}

// parameterValidationPath writes an error path relative to the string held by
// a generated validator parameter.
func parameterValidationPath(parameter string) validationPath {
	panic("excised: parameterValidationPath")
}

// child returns the context used for a field or collection value below c.
func (p validationPath) child(prefix string) validationPath {
	panic("excised: validationPath.child")
}

// renderValidationPath returns the Go expression passed to a generated
// validation error.
func renderValidationPath(path validationPath) string {
	panic("excised: renderValidationPath")
}

// hasValidations reports whether validating ut can write any code with the Go
// layout described by attCtx.
func hasValidations(attCtx *AttributeContext, ut expr.UserType) bool {
	panic("excised: hasValidations")
}

// toSlice returns Go code that represents the given slice.
func toSlice(val []any) string {
	panic("excised: toSlice")
}

// oneof produces code that compares target with each element of vals and ORs
// the result, e.g. "target == 1 || target == 2".
func oneof(target string, vals []any) string {
	panic("excised: oneof")
}

// constant returns the Go constant name of the format with the given value.
func constant(formatName string) string {
	panic("excised: constant")
}
