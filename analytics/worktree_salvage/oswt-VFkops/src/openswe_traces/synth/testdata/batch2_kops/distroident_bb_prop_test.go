package distributions_test

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"example.internal/kops/util/pkg/distributions"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func writeOSRelease(t *testing.T, contents string) string {
	t.Helper()
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(etc, "os-release"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

var exactTable = []struct {
	id, version string
	d           distributions.Distribution
}{
	{"amzn", "2023", distributions.DistributionAmazonLinux2023},
	{"amzn", "2027", distributions.DistributionAmazonLinux2027},
	{"debian", "11", distributions.DistributionDebian11},
	{"debian", "12", distributions.DistributionDebian12},
	{"debian", "13", distributions.DistributionDebian13},
	{"fedora", "41", distributions.DistributionFedora41},
	{"fedora", "42", distributions.DistributionFedora42},
	{"fedora", "43", distributions.DistributionFedora43},
	{"fedora", "44", distributions.DistributionFedora44},
	{"ubuntu", "22.04", distributions.DistributionUbuntu2204},
	{"ubuntu", "24.04", distributions.DistributionUbuntu2404},
	{"ubuntu", "25.10", distributions.DistributionUbuntu2510},
	{"ubuntu", "26.04", distributions.DistributionUbuntu2604},
}

// Detail 1: only ID= and VERSION_ID= lines are parsed; other fields ignored.
func TestDetail01_OnlyIDAndVersionIDParsed(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		row := exactTable[rng.Intn(len(exactTable))]
		// Noise fields that look relevant but must be ignored.
		noise := []string{
			`NAME="` + row.id + `junk"`,
			`PRETTY_NAME="` + row.id + ` ` + row.version + `.9"`,
			`ID_LIKE=debian`,
			`VERSION="` + row.version + ` extra"`,
			`VERSION_CODENAME=zzz`,
			fmt.Sprintf("CPE_NAME=cpe:/o:%s:%s:%d", row.id, row.version, rng.Intn(9)),
		}
		rng.Shuffle(len(noise), func(a, b int) { noise[a], noise[b] = noise[b], noise[a] })
		lines := append([]string{}, noise...)
		pad := func() string {
			if rng.Intn(3) == 0 {
				return "  "
			}
			return ""
		}
		pos := rng.Intn(len(lines) + 1)
		lines = append(lines[:pos], append([]string{pad() + `ID="` + row.id + `"`}, lines[pos:]...)...)
		pos = rng.Intn(len(lines) + 1)
		lines = append(lines[:pos], append([]string{pad() + `VERSION_ID="` + row.version + `"`}, lines[pos:]...)...)
		root := writeOSRelease(t, strings.Join(lines, "\n")+"\n")
		d, err := distributions.FindDistribution(root)
		if err != nil {
			t.Fatalf("i=%d unexpected error %v (content %q)", i, err, strings.Join(lines, "|"))
		}
		if d != row.d {
			t.Fatalf("i=%d got %v want %v", i, d, row.d)
		}
	}
}

// Detail 2: values get double-quotes stripped; unquoted values still work.
func TestDetail02_DoubleQuotesStripped(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		row := exactTable[rng.Intn(len(exactTable))]
		quoted := rng.Intn(2) == 0
		var id, ver string
		if quoted {
			id, ver = `"`+row.id+`"`, `"`+row.version+`"`
		} else {
			id, ver = row.id, row.version
		}
		root := writeOSRelease(t, "ID="+id+"\nVERSION_ID="+ver+"\n")
		d, err := distributions.FindDistribution(root)
		if err != nil {
			t.Fatalf("i=%d quoted=%v error %v", i, quoted, err)
		}
		if d != row.d {
			t.Fatalf("i=%d got %v want %v", i, d, row.d)
		}
	}
}

// Detail 3: identity key is <ID>-<VERSION_ID> exact match for the supported
// version list; nearby-but-unsupported versions fail.
func TestDetail03_ExactKeyMatchPerVersion(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		row := exactTable[rng.Intn(len(exactTable))]
		root := writeOSRelease(t, `ID="`+row.id+`"`+"\n"+`VERSION_ID="`+row.version+`"`+"\n")
		d, err := distributions.FindDistribution(root)
		if err != nil || d != row.d {
			t.Fatalf("i=%d %s-%s -> %v err %v", i, row.id, row.version, d, err)
		}
	}
	// Unsupported exact keys must error.
	bad := [][2]string{
		{"debian", "10"}, {"debian", "14"}, {"ubuntu", "20.04"}, {"ubuntu", "23.10"},
		{"fedora", "40"}, {"fedora", "45"}, {"amzn", "2"}, {"amzn", "2024"},
	}
	for i, kv := range bad {
		root := writeOSRelease(t, `ID="`+kv[0]+`"`+"\n"+`VERSION_ID="`+kv[1]+`"`+"\n")
		if d, err := distributions.FindDistribution(root); err == nil {
			t.Fatalf("i=%d unsupported %s-%s accepted as %v", i, kv[0], kv[1], d)
		}
	}
}

// Detail 4: prefix rules — cos-*, flatcar-*, centos-9.*, centos-10.*,
// rhel-8./9./10.*, rocky-8./9./10.* match any version under the prefix.
func TestDetail04_PrefixRules(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	type pref struct {
		id, verPrefix string
	}
	prefixes := []struct {
		id, verPrefix string
		d             distributions.Distribution
	}{
		{"cos", "cos-", distributions.DistributionContainerOS},
		{"flatcar", "flatcar-", distributions.DistributionFlatcar},
		{"centos", "centos-9.", distributions.DistributionCentOS9},
		{"centos", "centos-10.", distributions.DistributionCentOS10},
		{"rhel", "rhel-8.", distributions.DistributionRhel8},
		{"rhel", "rhel-9.", distributions.DistributionRhel9},
		{"rhel", "rhel-10.", distributions.DistributionRhel10},
		{"rocky", "rocky-8.", distributions.DistributionRocky8},
		{"rocky", "rocky-9.", distributions.DistributionRocky9},
		{"rocky", "rocky-10.", distributions.DistributionRocky10},
	}
	_ = pref{}
	for i := 0; i < 150; i++ {
		p := prefixes[rng.Intn(len(prefixes))]
		var ver string
		if p.id == "cos" || p.id == "flatcar" {
			ver = fmt.Sprintf("%d.%d.%d", rng.Intn(200), rng.Intn(100), rng.Intn(100))
		} else {
			ver = strings.TrimPrefix(p.verPrefix, p.id+"-")
			ver += fmt.Sprintf("%d.%d", rng.Intn(20), rng.Intn(20))
		}
		root := writeOSRelease(t, `ID="`+p.id+`"`+"\n"+`VERSION_ID="`+ver+`"`+"\n")
		d, err := distributions.FindDistribution(root)
		if err != nil || d != p.d {
			t.Fatalf("i=%d %s-%s -> %v err %v want %v", i, p.id, ver, d, err, p.d)
		}
	}
}

// Detail 5: the trailing dot is literal — bare centos-9 (no minor) is
// unsupported; likewise bare rhel-8 / rocky-10.
func TestDetail05_TrailingDotLiteral(t *testing.T) {
	for _, kv := range [][2]string{
		{"centos", "9"}, {"centos", "10"},
		{"rhel", "8"}, {"rhel", "9"}, {"rhel", "10"},
		{"rocky", "8"}, {"rocky", "9"}, {"rocky", "10"},
		{"centos", "9x"}, {"rhel", "8x"},
	} {
		root := writeOSRelease(t, `ID="`+kv[0]+`"`+"\n"+`VERSION_ID="`+kv[1]+`"`+"\n")
		if d, err := distributions.FindDistribution(root); err == nil {
			t.Fatalf("bare %s-%s unexpectedly accepted as %v", kv[0], kv[1], d)
		}
	}
}

// Detail 6: exact table is checked BEFORE prefix rules — an exact row wins.
func TestDetail06_ExactBeatsPrefix(t *testing.T) {
	// amzn-2023 is exact; even if a hypothetical amzn- prefix rule existed it
	// must not shadow it. The observable check: every exact row still maps to
	// its constant even with extra content that might confuse prefix scans.
	for _, row := range exactTable {
		root := writeOSRelease(t, `ID="`+row.id+`"`+"\n"+`VERSION_ID="`+row.version+`"`+"\n")
		d, err := distributions.FindDistribution(root)
		if err != nil || d != row.d {
			t.Fatalf("%s-%s -> %v err %v", row.id, row.version, d, err)
		}
	}
}

// Detail 7: unreadable/missing file -> "reading /etc/os-release: %v" wrap.
func TestDetail07_MissingFileError(t *testing.T) {
	root := t.TempDir() // no etc/os-release
	_, err := distributions.FindDistribution(root)
	if err == nil {
		t.Fatalf("missing file accepted")
	}
	if !strings.Contains(err.Error(), "reading /etc/os-release") {
		t.Fatalf("error %q lacks reading-wrap prefix", err.Error())
	}
	// directory where file should be is also an error
	if err := os.MkdirAll(filepath.Join(root, "etc", "os-release"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := distributions.FindDistribution(root); err == nil ||
		!strings.Contains(err.Error(), "reading /etc/os-release") {
		t.Fatalf("dir-as-file error = %v", err)
	}
}

// Detail 8: unknown distro -> error "unsupported distro %q" with the joined key.
func TestDetail08_UnsupportedDistroErrorFormat(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		id := fmt.Sprintf("nodistro%d", rng.Intn(10000))
		ver := fmt.Sprintf("%d.%d", rng.Intn(99), rng.Intn(99))
		root := writeOSRelease(t, `ID="`+id+`"`+"\n"+`VERSION_ID="`+ver+`"`+"\n")
		_, err := distributions.FindDistribution(root)
		if err == nil {
			t.Fatalf("i=%d unknown %s-%s accepted", i, id, ver)
		}
		want := fmt.Sprintf(`unsupported distro "%s-%s"`, id, ver)
		if err.Error() != want {
			t.Fatalf("i=%d error %q want %q", i, err.Error(), want)
		}
	}
}

// Detail 9: empty os-release / missing keys yield the joined key with empty
// halves — both missing gives `unsupported distro "-"`.
func TestDetail09_MissingKeysYieldDashKey(t *testing.T) {
	cases := []struct {
		contents string
		want     string
	}{
		{"", `unsupported distro "-"`},
		{"NAME=\"something\"\n", `unsupported distro "-"`},
		{"OTHER=x\nNAME=y\n", `unsupported distro "-"`},
		{"ID=\"debian\"\n", `unsupported distro "debian-"`},
		{"VERSION_ID=\"12\"\n", `unsupported distro "-12"`},
	}
	for i, c := range cases {
		root := writeOSRelease(t, c.contents)
		_, err := distributions.FindDistribution(root)
		if err == nil {
			t.Fatalf("i=%d contents %q accepted", i, c.contents)
		}
		if err.Error() != c.want {
			t.Fatalf("i=%d contents %q error %q want %q", i, c.contents, err.Error(), c.want)
		}
	}
}
