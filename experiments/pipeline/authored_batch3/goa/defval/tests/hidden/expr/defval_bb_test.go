package expr_test

// Hidden black-box suite for unit "defval".
// TestDetailNN numbers match DETAILS.md lines 1..12.
// Inferable:no lines assert shape only (an error exists, names the
// offending value/field), never the committed literal.

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "example.internal/apikit/v3/dsl"
	"example.internal/apikit/v3/expr"
)

// Detail 1 (Inferable: partially): absent defaults produce no errors; a
// present default is checked; named types are not re-descended (an
// invalid nested default inside a named type reports once no matter how
// many times the type is used).
func TestDetail01(t *testing.T) {
	// no authored default: clean
	expr.RunDSL(t, func() {
		Type("BbCleanT", func() { Attribute("a", String) })
	})

	// present invalid default: an error is produced
	err := expr.RunInvalidDSL(t, func() {
		Type("BbBadT", func() {
			Attribute("a", Int, func() { Default("bb-not-an-int") })
		})
	})
	require.NotEmpty(t, err.Error())

	// descent stops at named types: Inner used twice still reports once
	err = expr.RunInvalidDSL(t, func() {
		Type("BbInner", func() {
			Attribute("bbuniqinner", Int, func() { Default("bad") })
		})
		Type("BbOuter", func() {
			Attribute("a", "BbInner")
			Attribute("b", "BbInner")
		})
	})
	require.Equal(t, 1, strings.Count(err.Error(), "bbuniqinner"),
		"named-type default must be validated once, got %q", err.Error())
}

// Detail 2 (Inferable: no): errors identify the nested position —
// assert the offending field name / key surfaces, not the label text.
func TestDetail02(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbNest", func() {
			Attribute("bbfield", func() {
				Attribute("bbinner", Int, func() { Default("bad") })
			})
		})
	})
	require.Contains(t, err.Error(), "bbinner", "error must name the offending field")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbMapT", func() {
			Attribute("m", MapOf(String, Int), func() {
				Default(map[string]any{"bbmapkey": "notint"})
			})
		})
	})
	require.Contains(t, err.Error(), "bbmapkey", "error must name the offending map entry")
}

// Detail 3 (Inferable: partially): a nil authored value is rejected
// naming nil; interfaces and pointers unwrap to the concrete value.
func TestDetail03(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbNilT", func() {
			Attribute("a", Int, func() { Default((*int)(nil)) })
		})
	})
	require.Contains(t, err.Error(), "nil", "typed-nil default must be rejected naming nil")

	// a pointer to a compatible value unwraps and is accepted
	v := 7
	expr.RunDSL(t, func() {
		Type("BbPtrT", func() {
			Attribute("a", Int, func() { Default(&v) })
		})
	})
}

// Detail 4 (Inferable: no): primitive compatibility is enforced — an
// incompatible default errors; a value whose Go type matches the
// declared struct:field:type meta is accepted through that escape.
func TestDetail04(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbStrT", func() {
			Attribute("a", String, func() { Default(123) })
		})
	})
	require.NotEmpty(t, err.Error())

	err = expr.RunInvalidDSL(t, func() {
		Type("BbIntT", func() {
			Attribute("a", Int, func() { Default("bb-text") })
		})
	})
	require.NotEmpty(t, err.Error())

	// custom field type escape: a []byte-based Go type as String default
	// is normally incompatible but accepted when struct:field:type names it
	err = expr.RunInvalidDSL(t, func() {
		Type("BbRawNoMeta", func() {
			Attribute("a", String, func() { Default(json.RawMessage(`"bb"`)) })
		})
	})
	require.NotEmpty(t, err.Error(), "non-string-kind default must be rejected without the meta")

	expr.RunDSL(t, func() {
		Type("BbRawMeta", func() {
			Attribute("a", String, func() {
				Meta("struct:field:type", "json.RawMessage", "encoding/json")
				Default(json.RawMessage(`"bb"`))
			})
		})
	})
}

// Detail 5 (Inferable: no): numeric defaults must fit the design
// primitive; non-finite floats rejected on both widths.
func TestDetail05(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbOvf32", func() {
			Attribute("a", Int32, func() { Default(int64(1) << 40) })
		})
	})
	require.NotEmpty(t, err.Error(), "int32 default must reject a value exceeding its bounds")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbOvfU", func() {
			Attribute("a", UInt32, func() { Default(-1) })
		})
	})
	require.NotEmpty(t, err.Error(), "uint32 default must reject a negative value")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbF32", func() {
			Attribute("a", Float32, func() { Default(1e300) })
		})
	})
	require.NotEmpty(t, err.Error(), "float32 default must reject a value exceeding MaxFloat32")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbNaN", func() {
			Attribute("a", Float64, func() { Default(math.NaN()) })
		})
	})
	require.NotEmpty(t, err.Error(), "NaN default must be rejected")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbInf32", func() {
			Attribute("a", Float32, func() { Default(math.Inf(1)) })
		})
	})
	require.NotEmpty(t, err.Error(), "+Inf default must be rejected")

	// in-range values pass on every width
	expr.RunDSL(t, func() {
		Type("BbFit", func() {
			Attribute("i32", Int32, func() { Default(100) })
			Attribute("u32", UInt32, func() { Default(100) })
			Attribute("f32", Float32, func() { Default(1.5) })
			Attribute("f64", Float64, func() { Default(1e300) })
		})
	})
}

// Detail 6 (Inferable: partially): kind-matched defaults — union wants a
// map, object wants map or struct, array wants a sequence, map wants a
// map. Assert accept-vs-reject per kind only.
func TestDetail06(t *testing.T) {
	// union with a sequence default is rejected
	err := expr.RunInvalidDSL(t, func() {
		Type("BbUnionSeq", func() {
			OneOf("u", func() {
				Attribute("b1", String)
				Attribute("b2", Int)
				Default([]any{"x"})
			})
		})
	})
	require.Error(t, err, "union default of wrong kind (sequence) must error")

	// object accepts a map default
	expr.RunDSL(t, func() {
		Type("BbObjMapInner", func() {
			Attribute("n", Int)
		})
		Type("BbObjMap", func() {
			Attribute("p", "BbObjMapInner", func() { Default(map[string]any{"n": 1}) })
		})
	})

	// object rejects a scalar default
	err = expr.RunInvalidDSL(t, func() {
		Type("BbObjScalarInner", func() {
			Attribute("n", Int)
		})
		Type("BbObjScalar", func() {
			Attribute("p", "BbObjScalarInner", func() { Default(42) })
		})
	})
	require.Error(t, err, "scalar default on an object type must error")

	// array requires a sequence
	expr.RunDSL(t, func() {
		Type("BbArrOk", func() {
			Attribute("a", ArrayOf(String), func() { Default([]any{"x", "y"}) })
		})
	})
	err = expr.RunInvalidDSL(t, func() {
		Type("BbArrBad", func() {
			Attribute("a", ArrayOf(String), func() { Default("bbnotarray") })
		})
	})
	require.Error(t, err, "non-sequence default on an array must error")

	// map requires a map
	err = expr.RunInvalidDSL(t, func() {
		Type("BbMapBad", func() {
			Attribute("m", MapOf(String, Int), func() { Default([]any{1}) })
		})
	})
	require.Error(t, err, "non-map default on a map must error")
}

// Detail 7 (Inferable: no): Any-typed defaults accept only JSON-shaped
// values; nils anywhere are fine.
func TestDetail07(t *testing.T) {
	expr.RunDSL(t, func() {
		Type("BbAnyOk", func() {
			Attribute("a", Any, func() {
				Default(map[string]any{"s": "x", "n": 3, "b": true, "f": 1.5,
					"list": []any{1, "two", nil}, "nilfield": nil})
			})
		})
	})

	err := expr.RunInvalidDSL(t, func() {
		Type("BbAnyStruct", func() {
			Attribute("a", Any, func() { Default(struct{ X int }{X: 1}) })
		})
	})
	require.NotEmpty(t, err.Error(), "struct default on Any must be rejected")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbAnyIntKeys", func() {
			Attribute("a", Any, func() { Default(map[int]string{1: "x"}) })
		})
	})
	require.NotEmpty(t, err.Error(), "non-string-keyed map on Any must be rejected")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbAnyNaN", func() {
			Attribute("a", Any, func() { Default(math.NaN()) })
		})
	})
	require.NotEmpty(t, err.Error(), "non-finite float on Any must be rejected")

	err = expr.RunInvalidDSL(t, func() {
		Type("BbAnyFunc", func() {
			Attribute("a", Any, func() { Default(func() {}) })
		})
	})
	require.NotEmpty(t, err.Error(), "func default on Any must be rejected")
}

// Detail 8 (Inferable: no): enum membership compares by the design
// primitive — numeric cross-width equality works; non-members error.
func TestDetail08(t *testing.T) {
	// enum values authored as int compare equal to an int32 default after
	// both sides convert to the design primitive's reflect type
	expr.RunDSL(t, func() {
		Type("BbEnumOk", func() {
			Attribute("a", Int32, func() {
				Enum(1, 2, 3)
				Default(int32(2))
			})
			Attribute("s", String, func() {
				Enum("bbx", "bby")
				Default("bby")
			})
		})
	})

	err := expr.RunInvalidDSL(t, func() {
		Type("BbEnumBad", func() {
			Attribute("a", Int32, func() {
				Enum(1, 2, 3)
				Default(int32(4))
			})
		})
	})
	require.NotEmpty(t, err.Error())

	err = expr.RunInvalidDSL(t, func() {
		Type("BbEnumBadS", func() {
			Attribute("s", String, func() {
				Enum("bbx", "bby")
				Default("bbz")
			})
		})
	})
	require.NotEmpty(t, err.Error())
}

// Detail 9 (Inferable: partially): format/pattern apply only to string
// defaults; min/max compare numerically; length validations measure
// string/array/map content. ASCII cases only — rune-vs-byte unpinned.
func TestDetail09(t *testing.T) {
	// format applies to string defaults
	err := expr.RunInvalidDSL(t, func() {
		Type("BbFmt", func() {
			Attribute("a", String, func() {
				Format(expr.FormatEmail)
				Default("bb-not-an-email")
			})
		})
	})
	require.NotEmpty(t, err.Error())

	// format does not fire on a non-string default: the only error is the
	// type mismatch, nothing about the format
	err = expr.RunInvalidDSL(t, func() {
		Type("BbFmtSkip", func() {
			Attribute("a", String, func() {
				Format(expr.FormatEmail)
				Default(5)
			})
		})
	})
	require.NotEmpty(t, err.Error())
	require.NotContains(t, err.Error(), "format", "format rule must not fire on a non-string default")
	require.NotContains(t, err.Error(), "email")

	// pattern applies to strings
	err = expr.RunInvalidDSL(t, func() {
		Type("BbPat", func() {
			Attribute("a", String, func() {
				Pattern("^bb+$")
				Default("nope")
			})
		})
	})
	require.NotEmpty(t, err.Error())

	err = expr.RunInvalidDSL(t, func() {
		Type("BbPatSkip", func() {
			Attribute("a", String, func() {
				Pattern("^bb+$")
				Default(9)
			})
		})
	})
	require.NotEmpty(t, err.Error())
	require.NotContains(t, err.Error(), "pattern", "pattern rule must not fire on a non-string default")

	// numeric bounds compare by value
	err = expr.RunInvalidDSL(t, func() {
		Type("BbMin", func() {
			Attribute("a", Int, func() {
				Minimum(5)
				Default(3)
			})
		})
	})
	require.NotEmpty(t, err.Error())

	expr.RunDSL(t, func() {
		Type("BbMinOk", func() {
			Attribute("a", Int, func() {
				Minimum(5)
				Default(7)
			})
		})
	})

	// string length (ASCII: runes == bytes)
	err = expr.RunInvalidDSL(t, func() {
		Type("BbLen", func() {
			Attribute("a", String, func() {
				MinLength(3)
				Default("bb")
			})
		})
	})
	require.NotEmpty(t, err.Error())

	// collection length counts elements
	err = expr.RunInvalidDSL(t, func() {
		Type("BbArrLen", func() {
			Attribute("a", ArrayOf(String), func() {
				MinLength(2)
				Default([]any{"bb"})
			})
		})
	})
	require.NotEmpty(t, err.Error())
}

// Detail 10 (Inferable: no): object defaults require design-name fields
// to satisfy required, reject unknown fields and non-string keys, and
// match struct literals through generated Go field names.
func TestDetail10(t *testing.T) {
	// missing required field errors, naming the field
	err := expr.RunInvalidDSL(t, func() {
		Type("BbObjInner", func() {
			Attribute("bbreq", String)
			Required("bbreq")
		})
		Type("BbObjUse", func() {
			Attribute("p", "BbObjInner", func() { Default(map[string]any{}) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbreq", "missing-required error must name the field")

	// unknown map key errors, naming the key
	err = expr.RunInvalidDSL(t, func() {
		Type("BbObjUnk", func() {
			Attribute("bbknown", String)
		})
		Type("BbObjUse2", func() {
			Attribute("p", "BbObjUnk", func() { Default(map[string]any{"bbstray": 1}) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bbstray", "unknown-field error must name the key")

	// non-string map keys are reported by type
	err = expr.RunInvalidDSL(t, func() {
		Type("BbObjKeyT", func() {
			Attribute("bbknown", String)
		})
		Type("BbObjUse3", func() {
			Attribute("p", "BbObjKeyT", func() { Default(map[int]any{4242: "x"}) })
		})
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "int", "non-string-key error must report the key type")

	// struct literal matches via generated Go field names
	type bbObjStruct struct{ Username string }
	expr.RunDSL(t, func() {
		Type("BbObjSt", func() {
			Attribute("username", String)
		})
		Type("BbObjStUse", func() {
			Attribute("p", "BbObjSt", func() { Default(bbObjStruct{Username: "bb"}) })
		})
	})

	// a struct literal missing a required design field errors
	err = expr.RunInvalidDSL(t, func() {
		Type("BbObjStReq", func() {
			Attribute("username", String)
			Required("username")
		})
		Type("BbObjStReqUse", func() {
			Attribute("p", "BbObjStReq", func() { Default(struct{ Other string }{}) })
		})
	})
	require.Error(t, err)

	// a matched field whose value violates its own contract errors
	err = expr.RunInvalidDSL(t, func() {
		Type("BbObjStType", func() {
			Attribute("username", String)
		})
		Type("BbObjStTypeUse", func() {
			Attribute("p", "BbObjStType", func() { Default(struct{ Username int }{Username: 1}) })
		})
	})
	require.Error(t, err)
}

// Detail 11 (Inferable: no): union defaults need the canonical envelope
// (the declared TypeKey/ValueKey pair — "type"/"value" by default per the
// Union struct docs), a discriminator naming a declared branch, and the
// branch's own contract on the value.
func TestDetail11(t *testing.T) {
	// well-formed envelope passes: the union attr carries the default on
	// the OneOf expression itself.
	expr.RunDSL(t, func() {
		Type("BbUHost", func() {
			OneOf("u", func() {
				Attribute("b1", String)
				Attribute("b2", Int)
				Default(map[string]any{"type": "b1", "value": "bbok"})
			})
		})
	})

	// wrong branch contract on the value errors
	err := expr.RunInvalidDSL(t, func() {
		Type("BbUHostBadV", func() {
			OneOf("u", func() {
				Attribute("b1", String)
				Attribute("b2", Int)
				Default(map[string]any{"type": "b1", "value": 5})
			})
		})
	})
	require.Error(t, err, "branch value violating the branch's contract must error")

	// discriminator naming an undeclared branch errors
	err = expr.RunInvalidDSL(t, func() {
		Type("BbUHostBadB", func() {
			OneOf("u", func() {
				Attribute("b1", String)
				Attribute("b2", Int)
				Default(map[string]any{"type": "bbnosuch", "value": "x"})
			})
		})
	})
	require.Error(t, err, "unknown branch name must error")

	// missing discriminator errors
	err = expr.RunInvalidDSL(t, func() {
		Type("BbUHostNoT", func() {
			OneOf("u", func() {
				Attribute("b1", String)
				Attribute("b2", Int)
				Default(map[string]any{"value": "x"})
			})
		})
	})
	require.Error(t, err, "envelope missing the type key must error")
}

// Detail 12 (Inferable: no): errors are reported in deterministic order —
// assert sorted-by-key ordering via message positions.
func TestDetail12(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Type("BbMapOrd", func() {
			Attribute("m", MapOf(String, Int), func() {
				Default(map[string]any{"zzk": "bad", "mmk": "bad", "aak": "bad"})
			})
		})
	})
	require.Error(t, err)
	msg := err.Error()
	ia, im, iz := strings.Index(msg, "aak"), strings.Index(msg, "mmk"), strings.Index(msg, "zzk")
	require.True(t, ia >= 0 && im > ia && iz > im,
		"map-entry errors must be ordered by printed key: %q", msg)

	err = expr.RunInvalidDSL(t, func() {
		Type("BbObjOrd", func() {
			Attribute("bbonly", String)
		})
		Type("BbObjOrdUse", func() {
			Attribute("p", "BbObjOrd", func() {
				Default(map[string]any{"zzf": 1, "aaf": 2})
			})
		})
	})
	require.Error(t, err)
	msg = err.Error()
	require.True(t, strings.Index(msg, "aaf") >= 0 && strings.Index(msg, "zzf") > strings.Index(msg, "aaf"),
		"unknown-field errors must be ordered by name: %q", msg)
}
