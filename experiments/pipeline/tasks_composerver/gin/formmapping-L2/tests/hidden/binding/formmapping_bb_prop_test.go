// Black-box property suite for the formmapping unit (package binding).
// Exported: MapFormWithTag; binders Form/Query/Header/Uri/FormPost/FormMultipart;
// ErrConvertMapStringSlice, ErrConvertToMapString; BindUnmarshaler.
// MapFormWithTag and binders ONLY (no mapping/mappingByPtr). Seed 20260919.
//
// Contract (contract.md) -> property coverage table:
//
//	C1  "scalar kinds from string values" -> TestFMScalarsRandomProperty (>=10k)
//	C2  "default= when absent or empty first value" -> TestFMContractTableProperty
//	C3  "- tag / unexported fields skipped" -> TestFMContractTableProperty
//	C4  "slices resize; arrays exact length" -> TestFMContractTableProperty
//	C5  "collection_format csv/ssv/tsv/pipes/multi" -> TestFMCollectionFormatProperty
//	C6  "time_format/utc/location/unix + duration" -> TestFMTimeDurationProperty
//	C7  "struct/map JSON text fields" -> TestFMStructMapProperty
//	C8  "pointers allocated when set; circular safe" -> TestFMPtrCircularProperty
//	C9  "UnmarshalParam and TextUnmarshaler parser tag" -> TestFMCustomUnmarshalProperty
//	C10 "Form/Query/Header/Uri/FormPost binders" -> TestFMBindersProperty
//	C11 "map[string]string last; map[string][]string copy" -> TestFMMapTargetsProperty
//	C12 adversarial unknown collection format / uintptr -> TestFMAdversarialProperty
//	C13 unseen random scalar+tag mixes -> TestFMUnseenRandomProperty
package binding_test

import (
	"bytes"
	"encoding"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.internal/httprouter/binding"
	"github.com/stretchr/testify/require"
)

const (
	fmSeed  = 20260919
	fmCases = 10000
)

type fmHex int

func (f *fmHex) UnmarshalParam(param string) error {
	v, err := strconv.ParseInt(param, 16, 64)
	if err != nil {
		return err
	}
	*f = fmHex(v)
	return nil
}

type fmTextHex int

func (f *fmTextHex) UnmarshalText(text []byte) error {
	v, err := strconv.ParseInt(string(text), 16, 64)
	if err != nil {
		return err
	}
	*f = fmTextHex(v)
	return nil
}

var _ encoding.TextUnmarshaler = (*fmTextHex)(nil)

type fmPath []string

func (p *fmPath) UnmarshalParam(param string) error {
	elems := strings.Split(param, "/")
	if len(elems) < 2 {
		return errors.New("invalid format")
	}
	*p = elems
	return nil
}

type fmOID [12]byte

func (o *fmOID) UnmarshalParam(param string) error {
	if len(param) != 24 {
		return errors.New("invalid format")
	}
	_, err := hex.Decode(o[:], []byte(param))
	return err
}

func fmForm(vals map[string][]string) map[string][]string {
	return vals
}

// TestFMContractTableProperty translates form_mapping_test.go table scenarios.
func TestFMContractTableProperty(t *testing.T) {
	// Base types
	for _, tc := range []struct {
		typ  reflect.Type
		in   string
		want any
	}{
		{reflect.TypeOf(int(0)), "9", int(9)},
		{reflect.TypeOf(int8(0)), "9", int8(9)},
		{reflect.TypeOf(uint16(0)), "9", uint16(9)},
		{reflect.TypeOf(true), "True", true},
		{reflect.TypeOf(float32(0)), "9.1", float32(9.1)},
		{reflect.TypeOf(""), "test", "test"},
	} {
		st := reflect.New(reflect.StructOf([]reflect.StructField{{
			Name: "F", Type: tc.typ, Tag: `form:"F"`,
		}})).Interface()
		require.NoError(t, binding.MapFormWithTag(st, fmForm(map[string][]string{"F": {tc.in}}), "form"))
		got := reflect.ValueOf(st).Elem().Field(0).Interface()
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("base %v: got %v want %v", tc.typ, got, tc.want)
		}
	}

	// default=
	var def struct {
		Str   string `form:",default=defaultVal"`
		Int   int    `form:",default=9"`
		Slice []int  `form:",default=9"`
		Array [1]int `form:",default=9"`
	}
	require.NoError(t, binding.MapFormWithTag(&def, fmForm(map[string][]string{}), "form"))
	if def.Str != "defaultVal" || def.Int != 9 || def.Slice[0] != 9 || def.Array[0] != 9 {
		t.Fatal("defaults")
	}
	var emptyDef struct {
		F string `form:"field,default=DefVal"`
	}
	require.NoError(t, binding.MapFormWithTag(&emptyDef, fmForm(map[string][]string{"field": {""}}), "form"))
	if emptyDef.F != "DefVal" {
		t.Fatal("empty to default")
	}

	// ignore / unexported
	var skip struct {
		A int `form:"A"`
		B int `form:"-"`
		b int `form:"b"`
	}
	require.NoError(t, binding.MapFormWithTag(&skip, fmForm(map[string][]string{"A": {"9"}, "B": {"9"}, "b": {"9"}}), "form"))
	if skip.A != 9 || skip.B != 0 || skip.b != 0 {
		t.Fatal("skip/unexported")
	}

	// slice / array
	var sl struct {
		Slice []int `form:"slice,default=9"`
	}
	require.NoError(t, binding.MapFormWithTag(&sl, fmForm(map[string][]string{}), "form"))
	require.NoError(t, binding.MapFormWithTag(&sl, fmForm(map[string][]string{"slice": {"3", "4"}}), "form"))
	if !reflect.DeepEqual(sl.Slice, []int{3, 4}) {
		t.Fatal("slice")
	}
	var arr struct {
		Array [2]int `form:"array,default=9"`
	}
	if err := binding.MapFormWithTag(&arr, fmForm(map[string][]string{}), "form"); err == nil {
		t.Fatal("array wrong default len")
	}
	require.NoError(t, binding.MapFormWithTag(&arr, fmForm(map[string][]string{"array": {"3", "4"}}), "form"))
	if arr.Array != [2]int{3, 4} {
		t.Fatal("array ok")
	}
	if err := binding.MapFormWithTag(&arr, fmForm(map[string][]string{"array": {"3"}}), "form"); err == nil {
		t.Fatal("array short")
	}

	// absent key leaves field
	var untouched struct {
		F string `form:"field,default=defVal"`
		G string `form:"gone"`
	}
	require.NoError(t, binding.MapFormWithTag(&untouched, fmForm(map[string][]string{}), "form"))
	if untouched.F != "defVal" || untouched.G != "" {
		t.Fatal("absent keys")
	}

	// unknown kind
	var bad struct {
		U uintptr `form:"U"`
	}
	if err := binding.MapFormWithTag(&bad, fmForm(map[string][]string{"U": {"1"}}), "form"); err == nil {
		t.Fatal("uintptr")
	}
}

// TestFMCollectionFormatProperty covers csv/ssv/tsv/pipes/multi and invalid format.
func TestFMCollectionFormatProperty(t *testing.T) {
	var s struct {
		SliceMulti []int  `form:"slice_multi" collection_format:"multi"`
		SliceCsv   []int  `form:"slice_csv" collection_format:"csv"`
		SliceSsv   []int  `form:"slice_ssv" collection_format:"ssv"`
		SliceTsv   []int  `form:"slice_tsv" collection_format:"tsv"`
		SlicePipes []int  `form:"slice_pipes" collection_format:"pipes"`
		ArrayCsv   [2]int `form:"array_csv" collection_format:"csv"`
	}
	in := fmForm(map[string][]string{
		"slice_multi": {"1", "2"},
		"slice_csv":   {"1,2"},
		"slice_ssv":   {"1 2"},
		"slice_tsv":   {"1\t2"},
		"slice_pipes": {"1|2"},
		"array_csv":   {"1,2"},
	})
	require.NoError(t, binding.MapFormWithTag(&s, in, "form"))
	want := []int{1, 2}
	for _, sl := range [][]int{s.SliceMulti, s.SliceCsv, s.SliceSsv, s.SliceTsv, s.SlicePipes} {
		if !reflect.DeepEqual(sl, want) {
			t.Fatalf("collection: %v", sl)
		}
	}
	if s.ArrayCsv != [2]int{1, 2} {
		t.Fatal("array csv")
	}
	var bad struct {
		SliceCsv []int `form:"slice_csv" collection_format:"xxx"`
	}
	if err := binding.MapFormWithTag(&bad, fmForm(map[string][]string{"slice_csv": {"1,2"}}), "form"); err == nil {
		t.Fatal("invalid collection_format")
	}
	var defs struct {
		SliceMulti []int `form:",default=1;2;3" collection_format:"multi"`
		SliceCsv   []int `form:",default=1;2;3" collection_format:"csv"`
	}
	require.NoError(t, binding.MapFormWithTag(&defs, fmForm(map[string][]string{}), "form"))
	if !reflect.DeepEqual(defs.SliceMulti, []int{1, 2, 3}) || !reflect.DeepEqual(defs.SliceCsv, []int{1, 2, 3}) {
		t.Fatal("default collection split")
	}
}

// TestFMTimeDurationProperty covers time and duration tags.
func TestFMTimeDurationProperty(t *testing.T) {
	time.Local, _ = time.LoadLocation("Europe/Berlin")
	var s struct {
		Time      time.Time
		LocalTime time.Time `time_format:"2006-01-02"`
		CSTTime   time.Time `time_format:"2006-01-02" time_location:"Asia/Shanghai"`
		UTCTime   time.Time `time_format:"2006-01-02" time_utc:"1"`
	}
	require.NoError(t, binding.MapFormWithTag(&s, fmForm(map[string][]string{
		"Time":      {"2019-01-20T16:02:58Z"},
		"LocalTime": {"2019-01-20"},
		"CSTTime":   {"2019-01-20"},
		"UTCTime":   {"2019-01-20"},
	}), "form"))
	if s.Time.Format(time.RFC3339) != "2019-01-20T16:02:58Z" {
		t.Fatal("time RFC3339")
	}
	var dur struct {
		D time.Duration `form:"duration"`
	}
	require.NoError(t, binding.MapFormWithTag(&dur, fmForm(map[string][]string{"duration": {"5s"}}), "form"))
	if dur.D != 5*time.Second {
		t.Fatal("duration")
	}
	var nano struct {
		CreateTime time.Time `form:"createTime" time_format:"unixNano"`
	}
	require.NoError(t, binding.MapFormWithTag(&nano, fmForm(map[string][]string{"createTime": {"   "}}), "form"))
	if !nano.CreateTime.IsZero() {
		t.Fatal("unixNano empty")
	}
}

// TestFMStructMapProperty decodes nested struct/map from JSON strings.
func TestFMStructMapProperty(t *testing.T) {
	var st struct {
		J struct {
			I int `form:"I"`
		} `form:"J"`
		M map[string]int `form:"M"`
	}
	require.NoError(t, binding.MapFormWithTag(&st, fmForm(map[string][]string{
		"J": {`{"I": 9}`},
		"M": {`{"one": 1}`},
	}), "form"))
	if st.J.I != 9 || st.M["one"] != 1 {
		t.Fatal("struct/map json")
	}
}

// TestFMPtrCircularProperty checks lazy pointer allocation and circular refs.
func TestFMPtrCircularProperty(t *testing.T) {
	type ptrItem struct {
		Key int64 `form:"key"`
	}
	type req struct {
		Items []*ptrItem `form:"items"`
	}
	var r0 req
	require.NoError(t, binding.MapFormWithTag(&r0, fmForm(map[string][]string{}), "form"))
	if r0.Items != nil {
		t.Fatal("nil slice untouched")
	}
	var r1 req
	require.NoError(t, binding.MapFormWithTag(&r1, fmForm(map[string][]string{"items": {`{"key": 1}`}}), "form"))
	if len(r1.Items) != 1 || r1.Items[0].Key != 1 {
		t.Fatal("ptr slice")
	}
	type circ struct {
		S *circ `form:"-"`
	}
	var c circ
	require.NoError(t, binding.MapFormWithTag(&c, fmForm(map[string][]string{}), "form"))
}

// TestFMCustomUnmarshalProperty exercises BindUnmarshaler and TextUnmarshaler parser tag.
func TestFMCustomUnmarshalProperty(t *testing.T) {
	var hex struct {
		Foo fmHex `form:"foo"`
	}
	require.NoError(t, binding.MapFormWithTag(&hex, fmForm(map[string][]string{"foo": {"f5"}}), "form"))
	if int(hex.Foo) != 0xf5 {
		t.Fatal("UnmarshalParam")
	}
	var txt struct {
		Field fmTextHex `form:"field,parser=encoding.TextUnmarshaler"`
	}
	require.NoError(t, binding.MapFormWithTag(&txt, fmForm(map[string][]string{"field": {"f5"}}), "form"))
	if int(txt.Field) != 0xf5 {
		t.Fatal("TextUnmarshaler")
	}
	var paths struct {
		FileData []fmPath `form:"path" collection_format:"csv"`
	}
	require.NoError(t, binding.MapFormWithTag(&paths, fmForm(map[string][]string{"path": {"bar/foo,bar/foo/spam"}}), "form"))
	if len(paths.FileData) != 2 {
		t.Fatal("custom path csv")
	}
}

// TestFMBindersProperty routes through exported Form/Query/Header/Uri/FormPost binders.
func TestFMBindersProperty(t *testing.T) {
	var formObj struct {
		F int `form:"field"`
	}
	req := httptest.NewRequest(http.MethodPost, "/?query=1", strings.NewReader("field=6"))
	req.Header.Set("Content-Type", binding.MIMEPOSTForm)
	require.NoError(t, binding.Form.Bind(req, &formObj))
	if formObj.F != 6 {
		t.Fatal("Form")
	}

	var postObj struct {
		F int `form:"field"`
	}
	req2 := httptest.NewRequest(http.MethodPost, "/?field=9", strings.NewReader("field=6"))
	req2.Header.Set("Content-Type", binding.MIMEPOSTForm)
	require.NoError(t, binding.FormPost.Bind(req2, &postObj))
	if postObj.F != 6 {
		t.Fatal("FormPost ignores query")
	}

	var q struct {
		F int `form:"field"`
	}
	req3 := httptest.NewRequest(http.MethodGet, "/?field=6", nil)
	require.NoError(t, binding.Query.Bind(req3, &q))
	if q.F != 6 {
		t.Fatal("Query")
	}

	var h struct {
		F int `header:"X-Field"`
	}
	req4 := httptest.NewRequest(http.MethodGet, "/", nil)
	req4.Header.Set("X-Field", "6")
	require.NoError(t, binding.Header.Bind(req4, &h))
	if h.F != 6 {
		t.Fatal("Header")
	}

	var u struct {
		F int `uri:"field"`
	}
	require.NoError(t, binding.Uri.BindUri(fmForm(map[string][]string{"field": {"6"}}), &u))
	if u.F != 6 {
		t.Fatal("Uri")
	}

	var ext struct {
		F int `externalTag:"field"`
	}
	require.NoError(t, binding.MapFormWithTag(&ext, fmForm(map[string][]string{"field": {"6"}}), "externalTag"))
	if ext.F != 6 {
		t.Fatal("MapFormWithTag custom tag")
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	require.NoError(t, mw.WriteField("foo", "bar"))
	require.NoError(t, mw.WriteField("bar", "foo"))
	require.NoError(t, mw.Close())
	req5 := httptest.NewRequest(http.MethodPost, "/", body)
	req5.Header.Set("Content-Type", mw.FormDataContentType())
	var mp struct {
		Foo string `form:"foo" binding:"required"`
		Bar string `form:"bar" binding:"required"`
	}
	require.NoError(t, binding.FormMultipart.Bind(req5, &mp))
	if mp.Foo != "bar" || mp.Bar != "foo" {
		t.Fatal("FormMultipart")
	}
}

// TestFMMapTargetsProperty checks map destination semantics and conversion errors.
func TestFMMapTargetsProperty(t *testing.T) {
	mss := map[string][]string{}
	require.NoError(t, binding.MapFormWithTag(&mss, fmForm(map[string][]string{"a": {"1", "2"}, "b": {"3"}}), "form"))
	if !reflect.DeepEqual(mss["a"], []string{"1", "2"}) {
		t.Fatal("map[string][]string")
	}
	ms := map[string]string{}
	require.NoError(t, binding.MapFormWithTag(&ms, fmForm(map[string][]string{"k": {"first", "last"}}), "form"))
	if ms["k"] != "last" {
		t.Fatal("map[string]string last value")
	}
	var bad map[string]int
	if err := binding.MapFormWithTag(&bad, fmForm(map[string][]string{"k": {"1"}}), "form"); err != binding.ErrConvertToMapString {
		t.Fatalf("map[string]int: %v", err)
	}
	_ = binding.ErrConvertMapStringSlice
	_ = binding.ErrConvertToMapString
}

// TestFMAdversarialProperty probes invalid custom parsers and OID layout.
func TestFMAdversarialProperty(t *testing.T) {
	var p struct {
		FileData fmPath `form:"path"`
	}
	if err := binding.MapFormWithTag(&p, fmForm(map[string][]string{"path": {"bad"}}), "form"); err == nil {
		t.Fatal("custom path error")
	}
	val := `664a062ac74a8ad104e0e80f`
	var oid struct {
		ID fmOID `form:"id"`
	}
	require.NoError(t, binding.MapFormWithTag(&oid, fmForm(map[string][]string{"id": {val}}), "form"))
}

// TestFMScalarsRandomProperty runs >=10k random scalar mappings via MapFormWithTag.
func TestFMScalarsRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(fmSeed))
	kinds := []reflect.Kind{reflect.Int, reflect.Int64, reflect.Uint, reflect.Bool, reflect.Float64, reflect.String}
	for i := 0; i < fmCases; i++ {
		k := kinds[rng.Intn(len(kinds))]
		var fieldType reflect.Type
		var in string
		switch k {
		case reflect.Int, reflect.Int64:
			v := rng.Intn(2000) - 1000
			fieldType = reflect.TypeOf(int(0))
			if k == reflect.Int64 {
				fieldType = reflect.TypeOf(int64(0))
			}
			in = strconv.Itoa(v)
		case reflect.Uint:
			v := rng.Intn(5000)
			fieldType = reflect.TypeOf(uint(0))
			in = strconv.Itoa(v)
		case reflect.Bool:
			fieldType = reflect.TypeOf(false)
			in = []string{"true", "false", "1", "0"}[rng.Intn(4)]
		case reflect.Float64:
			v := float64(rng.Intn(10000)) / 100.0
			fieldType = reflect.TypeOf(float64(0))
			in = strconv.FormatFloat(v, 'f', 2, 64)
		case reflect.String:
			fieldType = reflect.TypeOf("")
			in = fmt.Sprintf("s%d", rng.Int63())
		}
		st := reflect.New(reflect.StructOf([]reflect.StructField{{
			Name: "F", Type: fieldType, Tag: `form:"F"`,
		}})).Interface()
		if err := binding.MapFormWithTag(st, fmForm(map[string][]string{"F": {in}}), "form"); err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		got := reflect.ValueOf(st).Elem().Field(0).Interface()
		// oracle via second bind into known struct type
		switch k {
		case reflect.Int:
			var o struct {
				F int `form:"F"`
			}
			require.NoError(t, binding.MapFormWithTag(&o, fmForm(map[string][]string{"F": {in}}), "form"))
			if o.F != got {
				t.Fatalf("case %d int", i)
			}
		case reflect.String:
			var o struct {
				F string `form:"F"`
			}
			require.NoError(t, binding.MapFormWithTag(&o, fmForm(map[string][]string{"F": {in}}), "form"))
			if o.F != in {
				t.Fatalf("case %d string", i)
			}
		}
	}
}

// TestFMUnseenRandomProperty mixes absent keys, defaults, and empty first values.
func TestFMUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(fmSeed + 1))
	for i := 0; i < 2000; i++ {
		mode := rng.Intn(3)
		var s struct {
			F string `form:"f,default=def"`
			G int    `form:"g"`
		}
		before := s
		m := fmForm(map[string][]string{})
		switch mode {
		case 0:
			m["f"] = []string{""}
		case 1:
			m["g"] = []string{strconv.Itoa(rng.Intn(100))}
		case 2:
			// absent
		}
		require.NoError(t, binding.MapFormWithTag(&s, m, "form"))
		if mode == 2 && s.G != 0 && s.F != "def" {
			t.Fatalf("case %d absent/default", i)
		}
		if mode == 0 && s.F != "def" {
			t.Fatalf("case %d empty default", i)
		}
		if mode == 2 && s.G != before.G && m["g"] == nil {
			// G should remain zero
		}
	}
}
