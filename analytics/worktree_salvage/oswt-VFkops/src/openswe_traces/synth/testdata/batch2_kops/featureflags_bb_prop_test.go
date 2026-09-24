package featureflag

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"testing"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

// Detail 1: item syntax "+Name"/"-Name"/"Name" (bare = enable).
func TestDetail01_ItemSyntax(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("ZZHidden%d", i)
		f := new(key, Bool(false))
		switch rng.Intn(3) {
		case 0:
			ParseFlags("+" + key)
		case 1:
			ParseFlags(key)
		case 2:
			ParseFlags("-Other" + key) // - on a different flag must not affect
			ParseFlags(key)
		}
		if !f.Enabled() {
			t.Fatalf("i=%d flag %s not enabled", i, key)
		}
		ParseFlags("-" + key)
		if f.Enabled() {
			t.Fatalf("i=%d flag %s still enabled after -", i, key)
		}
	}
}

// Detail 2: per-item TrimSpace and empty-item skip — " , ," is a no-op.
func TestDetail02_TrimSpaceAndEmptyItems(t *testing.T) {
	f := new("ZZHiddenTrim", Bool(false))
	ParseFlags(" , ,")
	if f.Enabled() {
		t.Fatalf("\" , ,\" mutated state")
	}
	ParseFlags("  +ZZHiddenTrim  ")
	if !f.Enabled() {
		t.Fatalf("padded +item not applied")
	}
	ParseFlags(" , -ZZHiddenTrim ,  ,")
	if f.Enabled() {
		t.Fatalf("padded -item not applied")
	}
}

// Detail 3: unknown flag names are logged and ignored — no error, and no
// flag is registered as a side effect.
func TestDetail03_UnknownFlagsIgnored(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("ZZNever%d%d", i, rng.Intn(1000))
		before := len(flags)
		ParseFlags("+" + name + ",-" + name + "X," + name + "Y")
		if _, ok := flags[name]; ok {
			t.Fatalf("i=%d unknown flag %q got registered", i, name)
		}
		if len(flags) != before {
			t.Fatalf("i=%d flag count changed %d->%d", i, before, len(flags))
		}
		if _, err := Get(name); err == nil {
			t.Fatalf("i=%d Get(%q) succeeded for unknown flag", i, name)
		}
	}
}

// Detail 4: Enabled precedence — explicit set beats default, default beats
// unset (false).
func TestDetail04_EnabledPrecedence(t *testing.T) {
	a := new("ZZPrecA", nil)
	if a.Enabled() {
		t.Fatalf("nil-default flag enabled")
	}
	b := new("ZZPrecB", Bool(true))
	if !b.Enabled() {
		t.Fatalf("true-default flag disabled")
	}
	c := new("ZZPrecC", Bool(false))
	if c.Enabled() {
		t.Fatalf("false-default flag enabled")
	}
	// Explicit set overrides default in both directions.
	ParseFlags("+ZZPrecC")
	if !c.Enabled() {
		t.Fatalf("explicit + failed to override false default")
	}
	ParseFlags("-ZZPrecB")
	if b.Enabled() {
		t.Fatalf("explicit - failed to override true default")
	}
}

// Detail 5: re-registration returns the same flag; a non-nil default is
// only installed when none was set.
func TestDetail05_Reregistration(t *testing.T) {
	f1 := new("ZZReg", Bool(true))
	f2 := new("ZZReg", Bool(false))
	if f1 != f2 {
		t.Fatalf("re-registration returned a different *FeatureFlag")
	}
	if f2.defaultValue == nil || !*f2.defaultValue {
		t.Fatalf("re-registration clobbered installed default")
	}
	// A flag created with nil default picks up a later non-nil default.
	g1 := new("ZZRegNil", nil)
	g2 := new("ZZRegNil", Bool(true))
	if g1 != g2 {
		t.Fatalf("re-registration returned different flag for nil-default")
	}
	if g2.defaultValue == nil || !*g2.defaultValue {
		t.Fatalf("non-nil default not installed on nil-default flag")
	}
}

// Detail 6: Get returns error "flag %s not found" for unregistered names.
func TestDetail06_GetNotFoundError(t *testing.T) {
	_, err := Get("ZZDefinitelyNotRegistered")
	if err == nil {
		t.Fatalf("Get on unregistered flag returned nil error")
	}
	want := "flag ZZDefinitelyNotRegistered not found"
	if err.Error() != want {
		t.Fatalf("Get error %q want %q", err.Error(), want)
	}
	// Registered flags are returned.
	f, err := Get("Spotinst")
	if err != nil || f == nil {
		t.Fatalf("Get(Spotinst)=%v err=%v", f, err)
	}
}

// Detail 7: explicit values persist across later ParseFlags calls until
// overwritten for that specific flag.
func TestDetail07_ExplicitPersistsAcrossCalls(t *testing.T) {
	f := new("ZZPersist", Bool(false))
	ParseFlags("+ZZPersist")
	ParseFlags("+Spotinst,-SpotinstOcean") // unrelated churn
	ParseFlags("")
	ParseFlags(" , , ")
	if !f.Enabled() {
		t.Fatalf("explicit + lost after unrelated ParseFlags calls")
	}
	ParseFlags("-ZZPersist,+Spotinst")
	if f.Enabled() {
		t.Fatalf("explicit - not applied")
	}
	if !Spotinst.Enabled() {
		t.Fatalf("unrelated flag not updated")
	}
}
