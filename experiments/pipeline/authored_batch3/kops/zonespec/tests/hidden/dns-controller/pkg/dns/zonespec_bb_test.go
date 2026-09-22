// Package dns_test is the hidden black-box suite for zonespec.
// One TestDetailNN per DETAILS.md commitment. Exported API only:
// ParseZoneSpec, ParseZoneRules, ZoneRules.MatchesExplicitly.
package dns_test

import (
	"testing"

	"example.internal/clustkit/dns-controller/pkg/dns"
	"example.internal/clustkit/dnsprovider/pkg/dnsprovider"
)

type fakeZone struct {
	name string
	id   string
}

func (z *fakeZone) Name() string { return z.name }
func (z *fakeZone) ID() string   { return z.id }
func (z *fakeZone) ResourceRecordSets() (dnsprovider.ResourceRecordSets, bool) {
	return nil, false
}

var _ dnsprovider.Zone = &fakeZone{}

// Detail 1 (Inferable: yes): bare name -> Name set with trailing dot, ID empty.
func TestDetail01(t *testing.T) {
	z, err := dns.ParseZoneSpec("example.com")
	if err != nil {
		t.Fatal(err)
	}
	if z.Name != "example.com." || z.ID != "" {
		t.Fatalf("%+v", z)
	}
}

// Detail 2 (Inferable: partially): `*/1234` selects by ID only.
func TestDetail02(t *testing.T) {
	z, err := dns.ParseZoneSpec("*/1234")
	if err != nil {
		t.Fatal(err)
	}
	if z.ID != "1234" || z.Name != "" {
		t.Fatalf("%+v", z)
	}
}

// Detail 3 (Inferable: yes): `name/1234` sets both.
func TestDetail03(t *testing.T) {
	z, err := dns.ParseZoneSpec("example.com/1234")
	if err != nil {
		t.Fatal(err)
	}
	if z.Name != "example.com." || z.ID != "1234" {
		t.Fatalf("%+v", z)
	}
}

// Detail 4 (Inferable: no): the spec splits on the FIRST `/` only.
func TestDetail04(t *testing.T) {
	z, err := dns.ParseZoneSpec("a/b/c")
	if err != nil {
		t.Fatal(err)
	}
	if z.Name != "a." || z.ID != "b/c" {
		t.Fatalf("%+v", z)
	}
}

// Detail 5 (Inferable: yes): input is whitespace-trimmed.
func TestDetail05(t *testing.T) {
	z, err := dns.ParseZoneSpec("  example.com  ")
	if err != nil {
		t.Fatal(err)
	}
	if z.Name != "example.com." {
		t.Fatalf("%+v", z)
	}
}

// Detail 6 (Inferable: partially): `*` and `*/*` set Wildcard and are not
// added to Zones.
func TestDetail06(t *testing.T) {
	for _, s := range []string{"*", "*/*"} {
		r, err := dns.ParseZoneRules([]string{s})
		if err != nil {
			t.Fatal(err)
		}
		if !r.Wildcard || len(r.Zones) != 0 {
			t.Fatalf("%q: wildcard=%v zones=%v", s, r.Wildcard, r.Zones)
		}
	}
	r, err := dns.ParseZoneRules([]string{"*", "example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.Wildcard || len(r.Zones) != 1 {
		t.Fatalf("mixed rules: %v %v", r.Wildcard, r.Zones)
	}
}

// Detail 7 (Inferable: partially): an empty rules list means Wildcard=true.
func TestDetail07(t *testing.T) {
	for _, zones := range [][]string{nil, {}} {
		r, err := dns.ParseZoneRules(zones)
		if err != nil {
			t.Fatal(err)
		}
		if !r.Wildcard {
			t.Fatalf("empty rules not wildcard: %+v", r)
		}
	}
}

// Detail 8 (Inferable: no): a rule whose Name mismatches is skipped (a later
// rule may still match); a rule whose ID mismatches vetoes the whole match.
func TestDetail08(t *testing.T) {
	zone := &fakeZone{name: "target.com", id: "y"}

	// name-mismatch rule is skipped; the matching rule still wins
	r, err := dns.ParseZoneRules([]string{"other.com", "target.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.MatchesExplicitly(zone) {
		t.Fatal("name-mismatch rule blocked a later match")
	}

	// id-mismatch rule vetoes immediately — the matching rule never reached
	r, err = dns.ParseZoneRules([]string{"*/zzz", "target.com"})
	if err != nil {
		t.Fatal(err)
	}
	if r.MatchesExplicitly(zone) {
		t.Fatal("id-mismatch rule did not veto")
	}
}

// Detail 9 (Inferable: yes): the zone's name is dot-suffixed before
// comparison; the id is compared verbatim.
func TestDetail09(t *testing.T) {
	r, err := dns.ParseZoneRules([]string{"example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.MatchesExplicitly(&fakeZone{name: "example.com", id: "x"}) {
		t.Fatal("un-suffixed zone name did not match")
	}
	if r.MatchesExplicitly(&fakeZone{name: "example.com.evil", id: "x"}) {
		t.Fatal("suffix-extended name matched")
	}
	r, err = dns.ParseZoneRules([]string{"*/1234"})
	if err != nil {
		t.Fatal(err)
	}
	if !r.MatchesExplicitly(&fakeZone{name: "z", id: "1234"}) {
		t.Fatal("verbatim id did not match")
	}
	if r.MatchesExplicitly(&fakeZone{name: "z", id: "12345"}) {
		t.Fatal("non-verbatim id matched")
	}
}

// Detail 10 (Inferable: partially): a wildcard ZoneRules never matches
// explicitly — only Zones entries are scanned.
func TestDetail10(t *testing.T) {
	for _, zones := range [][]string{nil, {"*"}, {"*/*"}} {
		r, err := dns.ParseZoneRules(zones)
		if err != nil {
			t.Fatal(err)
		}
		if !r.Wildcard {
			t.Fatalf("%v not wildcard", zones)
		}
		if r.MatchesExplicitly(&fakeZone{name: "anything.com", id: "1"}) {
			t.Fatalf("%v matched explicitly", zones)
		}
	}
}
