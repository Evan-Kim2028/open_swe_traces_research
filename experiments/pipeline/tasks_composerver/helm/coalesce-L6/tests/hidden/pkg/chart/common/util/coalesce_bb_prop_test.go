// Black-box property suite for the coalesce unit.
// Exported API only (api.md): CoalesceValues, MergeValues, CoalesceTables, MergeTables.
// Seed 20260919; >=10k cases; contract + coalesce_test.go coverage.
package util_test

import (
	"bytes"
	"fmt"
	"maps"
	"math/rand"
	"reflect"
	"testing"
	"text/template"

	"example.internal/chartkit/v4/pkg/chart/common"
	util "example.internal/chartkit/v4/pkg/chart/common/util"
	chart "example.internal/chartkit/v4/pkg/chart/v2"
)

const (
	bbSeed  = 20260919
	bbCases = 10000
)

var bbCoalesceYAML = []byte(`
top: yup
bottom: null
right: Null
left: NULL
front: ~
back: ""
nested:
  boat: null

global:
  name: Ishmael
  subject: Queequeg
  nested:
    boat: true

pequod:
  boat: null
  global:
    name: Stinky
    harpooner: Tashtego
    nested:
      boat: false
      sail: true
      foo2: null
  ahab:
    scope: whale
    boat: null
    nested:
      foo: true
      boat: null
    object: null
`)

func bbWithDeps(c *chart.Chart, deps ...*chart.Chart) *chart.Chart {
	c.AddDependency(deps...)
	return c
}

func bbTpl(tpl string, v map[string]any) (string, error) {
	var b bytes.Buffer
	tt := template.Must(template.New("t").Parse(tpl))
	if err := tt.Execute(&b, v); err != nil {
		return "", err
	}
	return b.String(), nil
}

func bbIsTable(v any) bool {
	_, ok := v.(map[string]any)
	return ok
}

// bbOracleCoalesceTables mirrors coalesce_test.go expectations (dst authoritative).
func bbOracleCoalesceTables(dst, src map[string]any) map[string]any {
	if src == nil {
		return dst
	}
	for key, srcVal := range src {
		dstVal, ok := dst[key]
		if !ok {
			if srcVal != nil {
				dst[key] = srcVal
			}
			continue
		}
		if dstVal == nil {
			delete(dst, key)
			continue
		}
		if bbIsTable(dstVal) && bbIsTable(srcVal) {
			bbOracleCoalesceTables(dstVal.(map[string]any), srcVal.(map[string]any))
			continue
		}
		// dst scalar/table wins on type mismatch
	}
	for k, v := range dst {
		if v == nil {
			delete(dst, k)
		} else if m, ok := v.(map[string]any); ok {
			bbCleanNilTables(m)
		}
	}
	return dst
}

func bbOracleMergeTables(dst, src map[string]any) map[string]any {
	if src == nil {
		return dst
	}
	for key, srcVal := range src {
		dstVal, ok := dst[key]
		if !ok {
			dst[key] = srcVal
			continue
		}
		if bbIsTable(dstVal) && bbIsTable(srcVal) {
			bbOracleMergeTables(dstVal.(map[string]any), srcVal.(map[string]any))
			continue
		}
		// dst wins on conflict
	}
	return dst
}

func bbCleanNilTables(m map[string]any) {
	for k, v := range m {
		if v == nil {
			delete(m, k)
		} else if sub, ok := v.(map[string]any); ok {
			bbCleanNilTables(sub)
		}
	}
}

func bbDeepCopy(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if sub, ok := v.(map[string]any); ok {
			out[k] = bbDeepCopy(sub)
		} else {
			out[k] = v
		}
	}
	return out
}

func bbMobyChart() *chart.Chart {
	return bbWithDeps(&chart.Chart{
		Metadata: &chart.Metadata{Name: "moby"},
		Values: map[string]any{
			"back": "exists", "bottom": "exists", "front": "exists", "left": "exists",
			"name": "moby", "nested": map[string]any{"boat": true}, "override": "bad",
			"right": "exists", "scope": "moby", "top": "nope",
			"global": map[string]any{"nested2": map[string]any{"l0": "moby"}},
			"pequod": map[string]any{
				"boat": "maybe",
				"ahab": map[string]any{
					"boat": "maybe", "nested": map[string]any{"boat": "maybe"},
				},
			},
		},
	},
		bbWithDeps(&chart.Chart{
			Metadata: &chart.Metadata{Name: "pequod"},
			Values: map[string]any{
				"name": "pequod", "scope": "pequod",
				"global": map[string]any{"nested2": map[string]any{"l1": "pequod"}},
				"boat":   false,
				"ahab":   map[string]any{"boat": false, "nested": map[string]any{"boat": false}},
			},
		},
			&chart.Chart{
				Metadata: &chart.Metadata{Name: "ahab"},
				Values: map[string]any{
					"global": map[string]any{
						"nested":  map[string]any{"foo": "bar", "foo2": "bar2"},
						"nested2": map[string]any{"l2": "ahab"},
					},
					"scope": "ahab", "name": "ahab", "boat": true,
					"nested": map[string]any{"foo": false, "boat": true},
					"object": map[string]any{"foo": "bar"},
				},
			},
		),
		&chart.Chart{
			Metadata: &chart.Metadata{Name: "spouter"},
			Values: map[string]any{
				"scope":  "spouter",
				"global": map[string]any{"nested2": map[string]any{"l1": "spouter"}},
			},
		},
	)
}

// ---------------------------------------------------------------------------
// contract table
// ---------------------------------------------------------------------------

func TestCoalesceContractTableProperty(t *testing.T) {
	// CoalesceTables fixture rows from coalesce_test.go
	dst := map[string]any{
		"name": "Ishmael",
		"address": map[string]any{
			"street": "123 Spouter Inn Ct.", "city": "Nantucket", "country": nil,
		},
		"details": map[string]any{"friends": []string{"Tashtego"}},
		"boat":    "pequod", "hole": nil,
	}
	src := map[string]any{
		"occupation": "whaler",
		"address":    map[string]any{"state": "MA", "street": "234 Spouter Inn Ct.", "country": "US"},
		"details":    "empty",
		"boat":       map[string]any{"mast": true},
		"hole":       "black",
	}
	want := bbDeepCopy(dst)
	bbOracleCoalesceTables(want, bbDeepCopy(src))
	got := bbDeepCopy(dst)
	util.CoalesceTables(got, bbDeepCopy(src))
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("CoalesceTables fixture mismatch\ngot  %v\nwant %v", got, want)
	}

	// MergeTables: nil keys preserved
	mdst := map[string]any{
		"name": "Ishmael",
		"address": map[string]any{
			"street": "123 Spouter Inn Ct.", "city": "Nantucket", "country": nil,
		},
		"details": map[string]any{"friends": []string{"Tashtego"}},
		"boat":    "pequod", "hole": nil,
	}
	msrc := bbDeepCopy(src)
	mwant := bbDeepCopy(mdst)
	bbOracleMergeTables(mwant, bbDeepCopy(msrc))
	mgot := bbDeepCopy(mdst)
	util.MergeTables(mgot, bbDeepCopy(msrc))
	if _, ok := mgot["hole"]; !ok {
		t.Fatal("MergeTables should preserve nil hole")
	}
	addr, _ := mgot["address"].(map[string]any)
	if _, ok := addr["country"]; !ok {
		t.Fatal("MergeTables should preserve nil country")
	}

	// CoalesceValues moby templates (subset of coalesce_test.go)
	c := bbMobyChart()
	vals, err := common.ReadValues(bbCoalesceYAML)
	if err != nil {
		t.Fatalf("read values: %v", err)
	}
	valsCopy := make(common.Values, len(vals))
	maps.Copy(valsCopy, vals)
	v, err := util.CoalesceValues(c, vals)
	if err != nil {
		t.Fatalf("CoalesceValues: %v", err)
	}
	for _, row := range []struct {
		tpl, expect string
	}{
		{"{{.top}}", "yup"},
		{"{{.global.name}}", "Ishmael"},
		{"{{.pequod.ahab.scope}}", "whale"},
		{"{{.pequod.ahab.global.harpooner}}", "Tashtego"},
	} {
		out, err := bbTpl(row.tpl, v)
		if err != nil || out != row.expect {
			t.Fatalf("tpl %q got %q want %q err=%v", row.tpl, out, row.expect, err)
		}
	}
	for _, k := range []string{"bottom", "right", "left", "front"} {
		if _, ok := v[k]; ok {
			t.Fatalf("null key %q should be removed", k)
		}
	}
	if !reflect.DeepEqual(valsCopy, vals) {
		t.Fatal("CoalesceValues mutated input vals")
	}

	mv, err := util.MergeValues(c, vals)
	if err != nil {
		t.Fatalf("MergeValues: %v", err)
	}
	for _, k := range []string{"bottom", "right", "left", "front"} {
		if _, ok := mv[k]; !ok {
			t.Fatalf("MergeValues should retain null key %q", k)
		}
	}

	// subchart nil cleanup (#31919)
	sub := &chart.Chart{
		Metadata: &chart.Metadata{Name: "child"},
		Values:   map[string]any{"keyMapping": map[string]any{"password": nil}},
	}
	parent := bbWithDeps(&chart.Chart{Metadata: &chart.Metadata{Name: "parent"}}, sub)
	cv, err := util.CoalesceValues(parent, map[string]any{})
	if err != nil {
		t.Fatalf("subchart nil clean: %v", err)
	}
	child, _ := cv["child"].(map[string]any)
	km, _ := child["keyMapping"].(map[string]any)
	if _, ok := km["password"]; ok {
		t.Fatal("nil default password should be cleaned")
	}

	// user null erases default
	sub2 := &chart.Chart{Metadata: &chart.Metadata{Name: "child"}, Values: map[string]any{"someKey": "default"}}
	parent2 := bbWithDeps(&chart.Chart{Metadata: &chart.Metadata{Name: "parent"}}, sub2)
	cv2, _ := util.CoalesceValues(parent2, map[string]any{"child": map[string]any{"someKey": nil}})
	child2, _ := cv2["child"].(map[string]any)
	if _, ok := child2["someKey"]; ok {
		t.Fatal("user null should erase subchart default")
	}
}

// ---------------------------------------------------------------------------
// table properties (>=10k)
// ---------------------------------------------------------------------------

func bbRandTable(rng *rand.Rand, depth int) map[string]any {
	m := map[string]any{}
	n := 1 + rng.Intn(4)
	for i := 0; i < n; i++ {
		k := fmt.Sprintf("k%d", i)
		switch rng.Intn(5) {
		case 0:
			m[k] = fmt.Sprintf("v%d", rng.Intn(20))
		case 1:
			m[k] = nil
		case 2:
			if depth > 0 {
				m[k] = bbRandTable(rng, depth-1)
			} else {
				m[k] = rng.Intn(100)
			}
		default:
			m[k] = rng.Intn(50)
		}
	}
	return m
}

func TestCoalesceTablesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed))
	for i := 0; i < bbCases; i++ {
		dst := map[string]any{
			"name": "Ishmael",
			"address": map[string]any{
				"street": "123 Spouter Inn Ct.", "city": "Nantucket", "country": nil,
			},
			"details": map[string]any{"friends": []string{"Tashtego"}},
			"boat":    "pequod", "hole": nil,
		}
		src := map[string]any{
			"occupation": "whaler",
			"address": map[string]any{
				"state": "MA", "street": "234 Spouter Inn Ct.", "country": "US",
			},
			"details": "empty",
			"boat":    map[string]any{"mast": true},
			"hole":    "black",
		}
		if rng.Intn(4) == 0 {
			src["extra"] = fmt.Sprintf("x%d", i%50)
		}
		want := bbDeepCopy(dst)
		bbOracleCoalesceTables(want, bbDeepCopy(src))
		got := bbDeepCopy(dst)
		got = util.CoalesceTables(got, src)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("case %d coalesce tables fixture", i)
		}
	}
}

func TestMergeTablesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 1))
	for i := 0; i < bbCases; i++ {
		dst := bbRandTable(rng, 2)
		src := bbRandTable(rng, 2)
		want := bbDeepCopy(dst)
		bbOracleMergeTables(want, bbDeepCopy(src))
		got := bbDeepCopy(dst)
		got = util.MergeTables(got, bbDeepCopy(src))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("case %d merge tables", i)
		}
	}
}

// ---------------------------------------------------------------------------
// chart-level properties
// ---------------------------------------------------------------------------

func bbSimpleChart(rng *rand.Rand) (*chart.Chart, map[string]any) {
	name := fmt.Sprintf("root-%d", rng.Intn(100))
	def := map[string]any{
		"alpha": fmt.Sprintf("def-%d", rng.Intn(10)),
		"beta":  rng.Intn(20),
	}
	ch := &chart.Chart{Metadata: &chart.Metadata{Name: name}, Values: def}
	vals := map[string]any{}
	if rng.Intn(2) == 0 {
		vals["alpha"] = fmt.Sprintf("usr-%d", rng.Intn(10))
	}
	return ch, vals
}

func TestCoalesceValuesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 2))
	for i := 0; i < bbCases; i++ {
		ch, vals := bbSimpleChart(rng)
		valsCopy := bbDeepCopy(vals)
		v, err := util.CoalesceValues(ch, vals)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if u, ok := vals["alpha"]; ok {
			if v["alpha"] != u {
				t.Fatalf("case %d user alpha override", i)
			}
		} else if fmt.Sprint(v["alpha"]) != fmt.Sprint(ch.Values["alpha"]) {
			t.Fatalf("case %d default alpha", i)
		}
		if u, ok := vals["beta"]; ok && u == nil {
			if _, present := v["beta"]; present {
				t.Fatalf("case %d user nil beta should remove", i)
			}
		}
		if !reflect.DeepEqual(valsCopy, vals) {
			t.Fatalf("case %d mutated vals", i)
		}
	}
}

func TestMergeValuesProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 3))
	for i := 0; i < bbCases; i++ {
		ch, vals := bbSimpleChart(rng)
		v, err := util.MergeValues(ch, vals)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		if u, ok := vals["alpha"]; ok && v["alpha"] != u {
			t.Fatalf("case %d merge alpha override", i)
		}
	}
}

// Adversarial: global propagation + partial override nil cleanup.
func TestCoalesceAdversarialProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 4))
	for i := 0; i < bbCases; i++ {
		sub := &chart.Chart{
			Metadata: &chart.Metadata{Name: "child"},
			Values: map[string]any{
				"ingress":    map[string]any{"feature": nil},
				"keyMapping": map[string]any{"password": nil, "format": "bcrypt"},
			},
		}
		parent := bbWithDeps(&chart.Chart{Metadata: &chart.Metadata{Name: "parent"}}, sub)
		vals := map[string]any{
			"global": map[string]any{"ingress": map[string]any{"feature": true}},
			"child": map[string]any{
				"keyMapping": map[string]any{"format": fmt.Sprintf("sha-%d", rng.Intn(9))},
			},
		}
		v, err := util.CoalesceValues(parent, vals)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		child, _ := v["child"].(map[string]any)
		ing, _ := child["ingress"].(map[string]any)
		if _, ok := ing["feature"]; ok {
			t.Fatalf("case %d subchart nil should not shadow global", i)
		}
		km, _ := child["keyMapping"].(map[string]any)
		if _, ok := km["password"]; ok {
			t.Fatalf("case %d nil password cleaned on partial override", i)
		}
	}
}

// Unseen-random keys outside moby fixture vocabulary.
func TestCoalesceUnseenRandomProperty(t *testing.T) {
	rng := rand.New(rand.NewSource(bbSeed + 5))
	lits := []string{"qzx", "wobble", "kraken", "nantucket", "ishmael"}
	for i := 0; i < bbCases; i++ {
		key := lits[rng.Intn(len(lits))]
		ch := &chart.Chart{
			Metadata: &chart.Metadata{Name: "x"},
			Values:   map[string]any{key: "default"},
		}
		vals := map[string]any{key: fmt.Sprintf("user-%d", i)}
		v, err := util.CoalesceValues(ch, vals)
		if err != nil || v[key] != vals[key] {
			t.Fatalf("case %d unseen key coalesce", i)
		}
	}
}
