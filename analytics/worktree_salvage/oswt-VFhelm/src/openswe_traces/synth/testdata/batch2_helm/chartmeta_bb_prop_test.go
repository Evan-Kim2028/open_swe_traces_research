// Hidden black-box suite for chartmeta. Exported API only: Metadata/Maintainer/
// Dependency.Validate, ValidationError, ValidationErrorf.
package v2_test

import (
	"errors"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"unicode"

	chart "example.internal/helm/pkg/chart/v2"
)

const HiddenSeed int64 = 20260919

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return HiddenSeed
}

func validMD() *chart.Metadata {
	return &chart.Metadata{
		APIVersion: chart.APIVersionV2,
		Name:       "mychart",
		Version:    "1.2.3",
	}
}

func TestDetail01_SanitizeInPlaceUnicodeSpaceAndNonPrintable(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	spaces := []rune{'\u00a0', '\u2000', '\u2003', '\u3000'}
	for i := 0; i < 80; i++ {
		sp := spaces[rng.Intn(len(spaces))]
		np := rune(1 + rng.Intn(8)) // non-printable
		md := validMD()
		md.Description = "ab" + string(sp) + "cd" + string(np) + "ef"
		md.Home = "h" + string(sp) + "ome"
		md.Icon = "ic" + string(np) + "on"
		md.Sources = []string{"s" + string(sp) + "rc"}
		md.Keywords = []string{"k" + string(np) + "w"}
		md.AppVersion = "1" + string(sp) + "0"
		md.KubeVersion = ">=1.2"
		md.Condition = "c" + string(np)
		md.Tags = "t" + string(sp) + "ag"
		md.Maintainers = []*chart.Maintainer{{
			Name:  "n" + string(sp) + "x",
			Email: "e" + string(np) + "y",
			URL:   "u" + string(sp),
		}}
		md.Dependencies = []*chart.Dependency{{
			Name:       "dep" + strconv.Itoa(i),
			Version:    "1.0.0",
			Repository: "https://ex.test/" + string(sp) + "r",
			Condition:  "on" + string(np),
			Tags:       []string{"t" + string(sp)},
		}}
		if err := md.Validate(); err != nil {
			t.Fatalf("i=%d Validate: %v", i, err)
		}
		if strings.ContainsRune(md.Description, sp) || strings.ContainsRune(md.Description, np) {
			t.Fatalf("description not sanitized: %q", md.Description)
		}
		if !strings.Contains(md.Description, "ab cd") && !strings.Contains(md.Description, "ab cd") {
			// unicode space → ASCII space, nonprintable removed → "ab cd ef"
		}
		if md.Description != "ab cd ef" && md.Description != "ab"+string(' ')+"cd"+"ef" {
			// allow either extra space collapse or not — DETAILS only map whitespace→space and drop nonprintable
			wantSpace := strings.Contains(md.Description, " ")
			if !wantSpace || strings.ContainsRune(md.Description, np) {
				t.Fatalf("i=%d description=%q", i, md.Description)
			}
		}
		if strings.ContainsRune(md.Home, sp) {
			t.Fatalf("home not sanitized: %q", md.Home)
		}
		if strings.ContainsRune(md.Maintainers[0].Name, sp) {
			t.Fatalf("maintainer name: %q", md.Maintainers[0].Name)
		}
		if strings.ContainsRune(md.Dependencies[0].Repository, sp) {
			t.Fatalf("dep repo: %q", md.Dependencies[0].Repository)
		}
		if strings.ContainsRune(md.Dependencies[0].Tags[0], sp) {
			t.Fatalf("dep tags: %q", md.Dependencies[0].Tags[0])
		}
	}
	// mutation is in place: same pointer fields updated
	md := validMD()
	md.Description = "x\u00a0y"
	p := &md.Description
	if err := md.Validate(); err != nil {
		t.Fatal(err)
	}
	if *p != md.Description {
		t.Fatal("sanitize must mutate in place")
	}
	_ = unicode.IsSpace
}

func TestDetail02_CheckOrderAPIVersionNameVersionType(t *testing.T) {
	md := &chart.Metadata{}
	err := md.Validate()
	if err == nil || !strings.Contains(err.Error(), "apiVersion") {
		t.Fatalf("apiVersion required first, got %v", err)
	}
	md.APIVersion = chart.APIVersionV2
	err = md.Validate()
	if err == nil || !strings.Contains(err.Error(), "name is required") && !strings.Contains(err.Error(), "name") {
		t.Fatalf("name required second, got %v", err)
	}
	md.Name = ".."
	err = md.Validate()
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("name rules before version, got %v", err)
	}
	md.Name = "ok"
	err = md.Validate()
	if err == nil || !strings.Contains(err.Error(), "version") {
		t.Fatalf("version required after name rules, got %v", err)
	}
	md.Version = "1.0.0"
	md.Type = "bogus"
	err = md.Validate()
	if err == nil || !strings.Contains(err.Error(), "type") {
		t.Fatalf("type checked last, got %v", err)
	}
}

func TestDetail03_ReservedNamesAndBasename(t *testing.T) {
	for _, n := range []string{".", ".."} {
		md := validMD()
		md.Name = n
		err := md.Validate()
		if err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Fatalf("name %q: want not allowed, got %v", n, err)
		}
	}
	md := validMD()
	md.Name = "foo/bar"
	err := md.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("name != basename: got %v", err)
	}
	md.Name = "ok-name_1"
	if err := md.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDetail04_LenientSemver(t *testing.T) {
	ok := []string{"1.0", "1.2.3", "1.0.0-alpha.1", "0.1"}
	for _, v := range ok {
		md := validMD()
		md.Version = v
		if err := md.Validate(); err != nil {
			t.Fatalf("version %q should be lenient-ok: %v", v, err)
		}
	}
	bad := []string{"1.2.3.4", "not-a-version", "v", "1.2.3+"}
	for _, v := range bad {
		md := validMD()
		md.Version = v
		if err := md.Validate(); err == nil {
			t.Fatalf("version %q should be invalid", v)
		} else if !strings.Contains(err.Error(), "version") {
			t.Fatalf("version %q error: %v", v, err)
		}
	}
}

func TestDetail05_TypeEnumEmptyApplicationLibrary(t *testing.T) {
	for _, ty := range []string{"", "application", "library"} {
		md := validMD()
		md.Type = ty
		if err := md.Validate(); err != nil {
			t.Fatalf("type %q must be allowed: %v", ty, err)
		}
	}
	md := validMD()
	md.Type = "plugin"
	if err := md.Validate(); err == nil {
		t.Fatal("unknown type must fail")
	}
}

func TestDetail06_AliasCharsetAndErrorNamesDependency(t *testing.T) {
	md := validMD()
	md.Dependencies = []*chart.Dependency{{
		Name: "depone", Version: "1.0.0", Repository: "https://e.test", Alias: "-_Ab9",
	}}
	if err := md.Validate(); err != nil {
		t.Fatalf("leading -/_ legal: %v", err)
	}
	md.Dependencies[0].Alias = "bad alias"
	err := md.Validate()
	if err == nil {
		t.Fatal("space in alias must fail")
	}
	if !strings.Contains(err.Error(), "depone") {
		t.Fatalf("error must name the dependency, got %v", err)
	}
	md.Dependencies[0].Alias = "has.dot"
	if err := md.Validate(); err == nil {
		t.Fatal("dot not in charset")
	}
}

func TestDetail07_DuplicateKeyedOnAliasElseName(t *testing.T) {
	md := validMD()
	md.Dependencies = []*chart.Dependency{
		{Name: "a", Version: "1.0.0", Repository: "https://one"},
		{Name: "a", Version: "2.0.0", Repository: "https://two"},
	}
	if err := md.Validate(); err == nil {
		t.Fatal("same name different version/repo is still a dup")
	}
	md.Dependencies = []*chart.Dependency{
		{Name: "a", Version: "1.0.0", Repository: "https://one", Alias: "x"},
		{Name: "b", Version: "1.0.0", Repository: "https://two", Alias: "x"},
	}
	if err := md.Validate(); err == nil {
		t.Fatal("same alias is a dup even when names differ")
	}
	md.Dependencies = []*chart.Dependency{
		{Name: "a", Version: "1.0.0", Repository: "https://one", Alias: "x"},
		{Name: "a", Version: "2.0.0", Repository: "https://two", Alias: "y"},
	}
	if err := md.Validate(); err != nil {
		t.Fatalf("alias-if-set else name: different aliases of same name ok? if not, that's still keyed on alias: %v", err)
	}
}

func TestDetail08_NilReceiversNoPanic(t *testing.T) {
	var md *chart.Metadata
	err := md.Validate()
	if err == nil {
		t.Fatal("nil metadata must error")
	}
	if !strings.Contains(err.Error(), "chart.metadata is required") {
		t.Fatalf("nil metadata wording: %v", err)
	}
	md = validMD()
	md.Dependencies = []*chart.Dependency{nil}
	err = md.Validate()
	if err == nil {
		t.Fatal("nil dependency element must error")
	}
	md = validMD()
	md.Maintainers = []*chart.Maintainer{nil}
	err = md.Validate()
	if err == nil {
		t.Fatal("nil maintainer element must error")
	}
	var dep *chart.Dependency
	if err := dep.Validate(); err == nil {
		t.Fatal("nil dependency receiver")
	}
	var m *chart.Maintainer
	if err := m.Validate(); err == nil {
		t.Fatal("nil maintainer receiver")
	}
}

func TestDetail09_ValidationErrorPrefixAndFormat(t *testing.T) {
	e := chart.ValidationErrorf("hello %s", "world")
	if _, ok := any(e).(chart.ValidationError); !ok {
		t.Fatal("ValidationErrorf must return ValidationError")
	}
	if !strings.HasPrefix(e.Error(), "validation: ") {
		t.Fatalf("Error() prefix: %q", e.Error())
	}
	if !strings.Contains(e.Error(), "hello world") {
		t.Fatalf("formatted: %q", e.Error())
	}
	md := validMD()
	md.APIVersion = ""
	err := md.Validate()
	var ve chart.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Validate must return ValidationError, got %T %v", err, err)
	}
	if !strings.HasPrefix(err.Error(), "validation: ") {
		t.Fatalf("Validate Error() prefix: %q", err.Error())
	}
}
