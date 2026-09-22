package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"example.internal/apikit/v3/expr"
)

// TestInitStructFieldsUsesFinalFieldTypeRef verifies that primitive alias
// casts use the exact type name selected during package planning.
func TestInitStructFieldsUsesFinalFieldTypeRef(t *testing.T) {
	alias := &expr.UserTypeExpr{
		TypeName: "Token",
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Meta: expr.MetaExpr{"struct:pkg:path": {"types"}},
		},
	}
	code, _, err := InitStructFields([]*InitArgData{{
		Name:         "token",
		Type:         expr.String,
		FieldName:    "Token",
		FieldType:    alias,
		FieldTypeRef: "types2.Token",
		FieldPointer: false,
		Pointer:      false,
	}}, "payload", "", "service")
	require.NoError(t, err)
	require.Equal(t, "payload.Token = types2.Token(token)\n", code)
}

// TestInitStructFieldsRequiresFinalFieldTypeRef verifies that plugin data
// cannot make the shared initializer guess a package name for a named field.
func TestInitStructFieldsRequiresFinalFieldTypeRef(t *testing.T) {
	alias := &expr.UserTypeExpr{
		TypeName: "Token",
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
		},
	}
	_, _, err := InitStructFields([]*InitArgData{{
		Name:      "token",
		Type:      expr.String,
		FieldName: "Token",
		FieldType: alias,
	}}, "payload", "", "service")
	require.EqualError(t, err, `initialize field "Token": missing final field type reference`)
}


func TestCamelCase(t *testing.T) {
	cases := map[string]struct {
		str        string
		firstUpper bool
		useAcronym bool
		expected   string
	}{
		"all lower":                     {"aaa", false, true, "aaa"},
		"all lower first upper":         {"aaa", true, true, "Aaa"},
		"start upper":                   {"Aaa", false, true, "aaa"},
		"mid upper":                     {"a_aa", false, true, "aAa"},
		"end upper":                     {"aa_a", false, true, "aaA"},
		"sequential uppers":             {"aa_aaaa", false, true, "aaAaaa"},
		"end sequential uppers":         {"aa_aa", false, true, "aaAa"},
		"multiple_uppers":               {"aa_aaa_aaa", false, true, "aaAaaAaa"},
		"underscores":                   {"aa_aaa_aaa", false, true, "aaAaaAaa"},
		"acronym":                       {"aa_id", false, true, "aaID"},
		"lower camel case":              {"aa_id", false, false, "aaId"},
		"upper camel case":              {"aaID", false, true, "aaID"},
		"lower camel case with acronym": {"aaId", false, true, "aaID"},

		"disable acronym":                    {"aa_id", false, false, "aaId"},
		"disable acronym first upper":        {"aaID", true, false, "AaId"},
		"disable acronym upper case acronym": {"aa_ID", false, false, "aaId"},
		"disable acronym upper camel case":   {"aaID", false, false, "aaId"},
		"disable acronym lower camel case":   {"aaId", false, false, "aaId"},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			actual := CamelCase(tc.str, tc.firstUpper, tc.useAcronym)
			if actual != tc.expected {
				t.Errorf("got %q, expected %q", actual, tc.expected)
			}
		})
	}
}

func TestProtobufNames(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "empty", want: "Val"},
		{name: "leading digits", source: "123_message", want: "_123Message"},
		{name: "acronym", source: "api_message", want: "APIMessage"},
		{name: "mixed Unicode", source: "café_message", want: "CafMessage"},
		{name: "only Unicode", source: "東京", want: "Val"},
		{name: "field keyword is a legal declaration", source: "string", want: "String"},
		{name: "invalid characters", source: "---", want: "Val"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := ProtobufName(test.source)
			if actual != test.want {
				t.Errorf("got %q, expected %q", actual, test.want)
			}
		})
	}
}

func TestProtobufFieldNames(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{name: "empty", want: "val"},
		{name: "leading digits", source: "123Field", want: "_123_field"},
		{name: "acronym", source: "HTTPServer", want: "http_server"},
		{name: "mixed Unicode", source: "caféField", want: "caf_field"},
		{name: "only Unicode", source: "東京", want: "val"},
		{name: "reserved word", source: "string", want: "string_"},
		{name: "invalid characters", source: "---", want: "val"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := ProtobufFieldName(test.source)
			if actual != test.want {
				t.Errorf("got %q, expected %q", actual, test.want)
			}
		})
	}
}


