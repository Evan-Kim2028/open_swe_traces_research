// Package distributions_test is the hidden black-box suite for distros.
// One TestDetailNN per DETAILS.md commitment. Only the exported distro vars
// and predicate methods are used.
package distributions_test

import (
	"testing"

	dist "example.internal/clustkit/util/pkg/distributions"
)

func allDistros() []dist.Distribution {
	return []dist.Distribution{
		dist.DistributionDebian11, dist.DistributionDebian12, dist.DistributionDebian13,
		dist.DistributionUbuntu2204, dist.DistributionUbuntu2404, dist.DistributionUbuntu2510,
		dist.DistributionUbuntu2604,
		dist.DistributionRhel8, dist.DistributionRhel9, dist.DistributionRhel10,
		dist.DistributionCentOS9, dist.DistributionCentOS10,
		dist.DistributionRocky8, dist.DistributionRocky9, dist.DistributionRocky10,
		dist.DistributionFedora41, dist.DistributionFedora42, dist.DistributionFedora43,
		dist.DistributionFedora44,
		dist.DistributionAmazonLinux2023, dist.DistributionAmazonLinux2027,
		dist.DistributionFlatcar, dist.DistributionContainerOS,
	}
}

func rpmDistros() []dist.Distribution {
	return []dist.Distribution{
		dist.DistributionRhel8, dist.DistributionRhel9, dist.DistributionRhel10,
		dist.DistributionCentOS9, dist.DistributionCentOS10,
		dist.DistributionRocky8, dist.DistributionRocky9, dist.DistributionRocky10,
		dist.DistributionFedora41, dist.DistributionFedora42, dist.DistributionFedora43,
		dist.DistributionFedora44,
		dist.DistributionAmazonLinux2023, dist.DistributionAmazonLinux2027,
	}
}

func debDistros() []dist.Distribution {
	return []dist.Distribution{
		dist.DistributionDebian11, dist.DistributionDebian12, dist.DistributionDebian13,
		dist.DistributionUbuntu2204, dist.DistributionUbuntu2404, dist.DistributionUbuntu2510,
		dist.DistributionUbuntu2604,
	}
}

// Detail 1 (Inferable: yes): packageFormat deb -> IsDebianFamily; rpm ->
// IsRHELFamily; immutable distros are neither.
func TestDetail01(t *testing.T) {
	for _, d := range debDistros() {
		if !d.IsDebianFamily() {
			t.Fatalf("deb distro not IsDebianFamily: %+v", d)
		}
		if d.IsRHELFamily() {
			t.Fatalf("deb distro IsRHELFamily: %+v", d)
		}
	}
	for _, d := range rpmDistros() {
		if !d.IsRHELFamily() {
			t.Fatalf("rpm distro not IsRHELFamily: %+v", d)
		}
		if d.IsDebianFamily() {
			t.Fatalf("rpm distro IsDebianFamily: %+v", d)
		}
	}
	for _, d := range []dist.Distribution{dist.DistributionFlatcar, dist.DistributionContainerOS} {
		if d.IsDebianFamily() || d.IsRHELFamily() {
			t.Fatalf("immutable distro in a package family: %+v", d)
		}
	}
}

// Detail 2 (Inferable: yes): project predicates are exact. Ubuntu is
// debian-family but not debian; amzn is rpm-family with its own predicate.
func TestDetail02(t *testing.T) {
	if !dist.DistributionDebian12.IsDebian() || !dist.DistributionDebian11.IsDebian() {
		t.Fatal("debian vars not IsDebian")
	}
	if dist.DistributionUbuntu2404.IsDebian() {
		t.Fatal("ubuntu reported IsDebian")
	}
	if !dist.DistributionUbuntu2404.IsUbuntu() || !dist.DistributionUbuntu2204.IsUbuntu() {
		t.Fatal("ubuntu vars not IsUbuntu")
	}
	if dist.DistributionDebian12.IsUbuntu() {
		t.Fatal("debian reported IsUbuntu")
	}
	if !dist.DistributionAmazonLinux2023.IsAmazonLinux() || !dist.DistributionAmazonLinux2027.IsAmazonLinux() {
		t.Fatal("amazonlinux vars not IsAmazonLinux")
	}
	if dist.DistributionRhel9.IsAmazonLinux() || dist.DistributionRhel9.IsDebian() || dist.DistributionRhel9.IsUbuntu() {
		t.Fatal("rhel reported a wrong project")
	}
}

// Detail 3 (Inferable: no): HasDNF gates per-project with version floors.
// Asserted shape: false for every non-rpm distro and the zero value; the kept
// rpm vars (all >= any floor) are true. The floors themselves and the
// unknown-project default are not pinnable black-box.
func TestDetail03(t *testing.T) {
	for _, d := range rpmDistros() {
		if !d.HasDNF() {
			t.Fatalf("rpm distro without dnf: %+v", d)
		}
	}
	for _, d := range append(debDistros(), dist.DistributionFlatcar, dist.DistributionContainerOS) {
		if d.HasDNF() {
			t.Fatalf("non-rpm distro with dnf: %+v", d)
		}
	}
	zero := dist.Distribution{}
	if zero.HasDNF() {
		t.Fatal("zero-value distro HasDNF")
	}
}

// Detail 4 (Inferable: no): IsSystemd is a constant true predicate.
func TestDetail04(t *testing.T) {
	for _, d := range allDistros() {
		if !d.IsSystemd() {
			t.Fatalf("distro not systemd: %+v", d)
		}
	}
}

// Detail 5 (Inferable: no): DefaultUsers returns a per-project user list; the
// specific lists are arbitrary so only shape is asserted: non-empty list of
// non-empty names, no error, deterministic. An unknown (zero-value) project
// must error.
func TestDetail05(t *testing.T) {
	// Only the projects the commitment lists may be asserted as succeeding;
	// which other projects are "known" is itself arbitrary.
	known := []dist.Distribution{
		dist.DistributionDebian12,
		dist.DistributionUbuntu2404,
		dist.DistributionCentOS9,
		dist.DistributionRhel9,
		dist.DistributionAmazonLinux2023,
		dist.DistributionRocky9,
		dist.DistributionFlatcar,
	}
	for _, d := range known {
		users, err := d.DefaultUsers()
		if err != nil {
			t.Fatalf("DefaultUsers error on %+v: %v", d, err)
		}
		if len(users) == 0 {
			t.Fatalf("DefaultUsers empty on %+v", d)
		}
		for _, u := range users {
			if u == "" {
				t.Fatalf("DefaultUsers empty name on %+v", d)
			}
		}
		again, err := d.DefaultUsers()
		if err != nil || len(again) != len(users) {
			t.Fatalf("DefaultUsers not deterministic on %+v", d)
		}
	}
	zero := dist.Distribution{}
	if users, err := zero.DefaultUsers(); err == nil {
		t.Fatalf("unknown project did not error (got %v)", users)
	}
}

// Detail 6 (Inferable: no): HasLoopbackEtcResolvConf is true for a fixed
// subset and otherwise probes the host fs. Asserted shape: never panics,
// deterministic, and at least one distro reports true (the special-cased set
// is non-empty).
func TestDetail06(t *testing.T) {
	anyTrue := false
	for _, d := range allDistros() {
		a := d.HasLoopbackEtcResolvConf()
		b := d.HasLoopbackEtcResolvConf()
		if a != b {
			t.Fatalf("non-deterministic result on %+v", d)
		}
		if a {
			anyTrue = true
		}
	}
	if !anyTrue {
		t.Fatal("no distro reported HasLoopbackEtcResolvConf")
	}
}

// Detail 7 (Inferable: yes): Version returns the project-scoped float.
func TestDetail07(t *testing.T) {
	cases := []struct {
		d    dist.Distribution
		want float32
	}{
		{dist.DistributionUbuntu2204, 22.04},
		{dist.DistributionUbuntu2404, 24.04},
		{dist.DistributionDebian12, 12},
		{dist.DistributionRhel9, 9},
		{dist.DistributionRhel10, 10},
		{dist.DistributionRocky8, 8},
		{dist.DistributionFedora43, 43},
		{dist.DistributionAmazonLinux2023, 2023},
		{dist.DistributionFlatcar, 0},
	}
	for _, c := range cases {
		if got := c.d.Version(); got != c.want {
			t.Fatalf("Version: got %v want %v", got, c.want)
		}
	}
}

// Detail 8 (Inferable: no): ForceNftables = rpm-family minus an allowlist.
// Asserted shape: false for every non-rpm distro and the zero value; the
// rpm partition is non-degenerate (at least one true AND at least one false).
// Allowlist membership is not pinned.
func TestDetail08(t *testing.T) {
	for _, d := range append(debDistros(), dist.DistributionFlatcar, dist.DistributionContainerOS) {
		if d.ForceNftables() {
			t.Fatalf("non-rpm distro forces nftables: %+v", d)
		}
	}
	zero := dist.Distribution{}
	if zero.ForceNftables() {
		t.Fatal("zero-value distro ForceNftables")
	}
	anyTrue, anyFalse := false, false
	for _, d := range rpmDistros() {
		if d.ForceNftables() {
			anyTrue = true
		} else {
			anyFalse = true
		}
	}
	if !anyTrue || !anyFalse {
		t.Fatalf("rpm partition degenerate: anyTrue=%v anyFalse=%v", anyTrue, anyFalse)
	}
}
