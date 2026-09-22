// Copyright 2026 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"testing"

	"github.com/google/go-github/v92/github"
)

func cp(vt github.PropertyValueType, dv any) github.CustomProperty {
	return github.CustomProperty{ValueType: vt, DefaultValue: dv}
}

// Detail 1: DefaultValueString succeeds for string, single_select and url
// value types when the payload holds a string; anything else returns false.
// Inferable: partially — the value-type set is named in the surviving doc
// comment; asserted for those three plus a wrong-type and wrong-shape
// refusal.
func TestDetail01(t *testing.T) {
	for _, vt := range []github.PropertyValueType{
		github.PropertyValueTypeString,
		github.PropertyValueTypeSingleSelect,
		github.PropertyValueTypeURL,
	} {
		got, ok := cp(vt, "value").DefaultValueString()
		if !ok || got != "value" {
			t.Errorf("ValueType %s with string payload: (%q, %v), want (\"value\", true)", vt, got, ok)
		}
	}
	if _, ok := cp(github.PropertyValueTypeMultiSelect, "value").DefaultValueString(); ok {
		t.Error("multi_select payload accepted by DefaultValueString")
	}
	if _, ok := cp(github.PropertyValueTypeString, 42).DefaultValueString(); ok {
		t.Error("non-string payload accepted by DefaultValueString")
	}
}

// Detail 2: DefaultValueStrings succeeds only for multi_select with a
// string-slice payload. Inferable: partially — asserted for a real []string;
// the []any tolerance is left unasserted.
func TestDetail02(t *testing.T) {
	got, ok := cp(github.PropertyValueTypeMultiSelect, []string{"a", "b"}).DefaultValueStrings()
	if !ok || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("multi_select []string payload: (%v, %v), want ([a b], true)", got, ok)
	}
	if _, ok := cp(github.PropertyValueTypeString, []string{"a"}).DefaultValueStrings(); ok {
		t.Error("string type with []string payload accepted by DefaultValueStrings")
	}
	if _, ok := cp(github.PropertyValueTypeMultiSelect, "a").DefaultValueStrings(); ok {
		t.Error("non-slice payload accepted by DefaultValueStrings")
	}
}

// Detail 3: DefaultValueBool succeeds only for true_false when the payload
// is a string parseable as a boolean. Inferable: partially — asserted for
// parseable and unparseable string payloads; the bool-payload edge is left
// unasserted.
func TestDetail03(t *testing.T) {
	got, ok := cp(github.PropertyValueTypeTrueFalse, "true").DefaultValueBool()
	if !ok || got != true {
		t.Errorf("true_false \"true\": (%v, %v), want (true, true)", got, ok)
	}
	got, ok = cp(github.PropertyValueTypeTrueFalse, "false").DefaultValueBool()
	if !ok || got != false {
		t.Errorf("true_false \"false\": (%v, %v), want (false, true)", got, ok)
	}
	if _, ok := cp(github.PropertyValueTypeTrueFalse, "notabool").DefaultValueBool(); ok {
		t.Error("unparseable string accepted by DefaultValueBool")
	}
	if _, ok := cp(github.PropertyValueTypeString, "true").DefaultValueBool(); ok {
		t.Error("string type accepted by DefaultValueBool")
	}
}

// Detail 4: every accessor reports (value, false) rather than guessing when
// the value type or payload shape does not match. (Inferable: yes)
func TestDetail04(t *testing.T) {
	for _, c := range []github.CustomProperty{
		cp(github.PropertyValueTypeString, nil),
		cp(github.PropertyValueTypeMultiSelect, nil),
		cp(github.PropertyValueTypeTrueFalse, nil),
	} {
		if _, ok := c.DefaultValueString(); ok {
			t.Errorf("ValueType %s nil payload: DefaultValueString ok", c.ValueType)
		}
		if _, ok := c.DefaultValueStrings(); ok {
			t.Errorf("ValueType %s nil payload: DefaultValueStrings ok", c.ValueType)
		}
		if _, ok := c.DefaultValueBool(); ok {
			t.Errorf("ValueType %s nil payload: DefaultValueBool ok", c.ValueType)
		}
	}
}
