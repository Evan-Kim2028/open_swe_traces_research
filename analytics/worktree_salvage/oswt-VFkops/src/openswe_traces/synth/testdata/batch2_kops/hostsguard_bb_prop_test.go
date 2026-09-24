package hosts

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func writeHosts(t *testing.T, lines []string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func readHosts(t *testing.T, p string) []string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.TrimSuffix(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func noopMutator(guarded []string) (*HostMap, error) {
	m := &HostMap{}
	m.Parse(guarded)
	return m, nil
}

// Detail 1: marker lines match on the whitespace-trimmed line text —
// padded markers still count.
func TestDetail01_PaddedMarkersCount(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 40; i++ {
		pad := strings.Repeat(" ", 1+rng.Intn(4))
		lines := []string{
			"127.0.0.1 localhost",
			pad + GUARD_BEGIN + pad,
			fmt.Sprintf("10.0.0.%d host%d", rng.Intn(200), i),
			pad + GUARD_END + pad,
			"127.0.0.2 other",
		}
		p := writeHosts(t, lines)
		err := UpdateHostsFileWithRecords(p, noopMutator)
		if err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		out := readHosts(t, p)
		joined := strings.Join(out, "\n")
		if !strings.Contains(joined, GUARD_BEGIN) || !strings.Contains(joined, GUARD_END) {
			t.Fatalf("i=%d markers missing in %q", i, joined)
		}
		if !strings.Contains(joined, "127.0.0.1 localhost") || !strings.Contains(joined, "127.0.0.2 other") {
			t.Fatalf("i=%d unmanaged lines lost in %q", i, joined)
		}
	}
}

// Detail 2: the begin-marker line itself is consumed into the managed set
// (not emitted verbatim to the mutator).
func TestDetail02_BeginMarkerConsumed(t *testing.T) {
	p := writeHosts(t, []string{
		"127.0.0.1 localhost",
		GUARD_BEGIN,
		"10.0.0.1 h1",
		GUARD_END,
	})
	var got []string
	err := UpdateHostsFileWithRecords(p, func(guarded []string) (*HostMap, error) {
		got = append([]string{}, guarded...)
		return noopMutator(guarded)
	})
	if err != nil {
		t.Fatal(err)
	}
	// The begin marker is consumed into the managed set — it reaches the
	// mutator as a guarded line, and the freshly rendered block replaces
	// everything wholesale.
	foundBegin, foundContent := false, false
	for _, l := range got {
		if strings.TrimSpace(l) == GUARD_BEGIN {
			foundBegin = true
		}
		if strings.Contains(l, "10.0.0.1 h1") {
			foundContent = true
		}
	}
	if !foundBegin {
		t.Fatalf("begin marker not in managed set: %v", got)
	}
	if !foundContent {
		t.Fatalf("managed content line missing from mutator input %v", got)
	}
}

// Detail 3: a begin marker seen while already inside the block only logs a
// warning; the line still joins the managed set.
func TestDetail03_NestedBeginJoinsManagedSet(t *testing.T) {
	p := writeHosts(t, []string{
		"127.0.0.1 localhost",
		GUARD_BEGIN,
		"10.0.0.1 h1",
		GUARD_BEGIN, // nested
		"10.0.0.2 h2",
		GUARD_END,
	})
	var got []string
	err := UpdateHostsFileWithRecords(p, func(guarded []string) (*HostMap, error) {
		got = append([]string{}, guarded...)
		return noopMutator(guarded)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 {
		t.Fatalf("nested begin dropped content: mutator got %v", got)
	}
}

// Detail 4: an end marker seen OUTSIDE a block is dropped from output.
func TestDetail04_StrayEndMarkerDropped(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 40; i++ {
		lines := []string{
			"127.0.0.1 localhost",
			GUARD_END, // stray, before any begin
			"127.0.0.2 other",
			GUARD_BEGIN,
			fmt.Sprintf("10.0.0.%d h%d", rng.Intn(200), i),
			GUARD_END,
			GUARD_END, // stray, after close
			"127.0.0.3 tail",
		}
		p := writeHosts(t, lines)
		if err := UpdateHostsFileWithRecords(p, noopMutator); err != nil {
			t.Fatalf("i=%d %v", i, err)
		}
		out := readHosts(t, p)
		count := 0
		for _, l := range out {
			if strings.TrimSpace(l) == GUARD_END {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("i=%d found %d end markers want 1: %v", i, count, out)
		}
		if !strings.Contains(strings.Join(out, "\n"), "127.0.0.3 tail") {
			t.Fatalf("i=%d tail line lost: %v", i, out)
		}
	}
}

// Detail 5: all lines inside a block go to the mutator and are replaced
// wholesale by the freshly rendered block — repeated/duplicated blocks
// collapse into one.
func TestDetail05_DuplicateBlocksCollapse(t *testing.T) {
	p := writeHosts(t, []string{
		"127.0.0.1 localhost",
		GUARD_BEGIN,
		"10.0.0.1 h1",
		GUARD_END,
		"127.0.0.2 mid",
		GUARD_BEGIN,
		"10.0.0.2 h2",
		GUARD_END,
		"127.0.0.3 tail",
	})
	var got []string
	err := UpdateHostsFileWithRecords(p, func(guarded []string) (*HostMap, error) {
		got = append([]string{}, guarded...)
		return noopMutator(guarded)
	})
	if err != nil {
		t.Fatal(err)
	}
	// Both managed blocks' content lands in one mutator call.
	if len(got) < 2 {
		t.Fatalf("expected both blocks' lines, got %v", got)
	}
	out := readHosts(t, p)
	var begins, ends int
	for _, l := range out {
		if strings.TrimSpace(l) == GUARD_BEGIN {
			begins++
		}
		if strings.TrimSpace(l) == GUARD_END {
			ends++
		}
	}
	if begins != 1 || ends != 1 {
		t.Fatalf("duplicate blocks not collapsed: %v", out)
	}
}

// Detail 6: Parse — empty lines skipped; comment lines skipped (the two
// marker lines only warn); any OTHER comment is a bad line; single-field
// non-comment lines are bad lines.
func TestDetail06_ParseBadLines(t *testing.T) {
	m := &HostMap{}
	bad := m.Parse([]string{
		"",
		"   ",
		"# a comment",
		GUARD_BEGIN,
		GUARD_END,
		"singlefield",
		"10.0.0.1 host1",
		"#another comment",
		"onefield",
	})
	// Bad lines: the two non-marker comments + the two single-field lines.
	wantBad := map[string]bool{
		"# a comment": true, "#another comment": true,
		"singlefield": true, "onefield": true,
	}
	for _, b := range bad {
		if !wantBad[b] {
			t.Fatalf("unexpected bad line %q", b)
		}
		delete(wantBad, b)
	}
	if len(wantBad) != 0 {
		t.Fatalf("missing bad lines %v (got %v)", wantBad, bad)
	}
	if len(m.records) != 1 || m.records[0].Hostname != "host1" || m.records[0].Address != "10.0.0.1" {
		t.Fatalf("records %v", m.records)
	}
}

// Detail 7: "addr h1 h2 ..." yields one record PER hostname field.
func TestDetail07_MultiHostnameLines(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 60; i++ {
		n := 2 + rng.Intn(4)
		hosts := make([]string, n)
		for j := range hosts {
			hosts[j] = fmt.Sprintf("h%d-%d", j, i)
		}
		m := &HostMap{}
		bad := m.Parse([]string{"10.0.0." + strconv.Itoa(1+i%200) + " " + strings.Join(hosts, " ")})
		if len(bad) != 0 {
			t.Fatalf("i=%d bad lines %v", i, bad)
		}
		if len(m.records) != n {
			t.Fatalf("i=%d records %v want %d", i, m.records, n)
		}
		for j, h := range hosts {
			if m.records[j].Hostname != h {
				t.Fatalf("i=%d record %d = %q want %q", i, j, m.records[j].Hostname, h)
			}
		}
	}
}

// Detail 8: ReplaceRecords — new records go FIRST, then retained
// non-matching records.
func TestDetail08_ReplaceRecordsPrepends(t *testing.T) {
	m := &HostMap{}
	m.Parse([]string{
		"10.0.0.1 alpha",
		"10.0.0.2 beta",
		"10.0.0.3 alpha",
	})
	m.ReplaceRecords("alpha", []string{"192.168.0.1"})
	if len(m.records) != 2 {
		t.Fatalf("records %v", m.records)
	}
	if m.records[0].Hostname != "alpha" || m.records[0].Address != "192.168.0.1" {
		t.Fatalf("new record not first: %v", m.records[0])
	}
	if m.records[1].Hostname != "beta" {
		t.Fatalf("retained record wrong: %v", m.records[1])
	}
}

// Detail 9: render — group by address, "addr\thost1 host2" sorted hosts,
// address lines sorted, exactly one blank line before the block, one
// trailing newline.
func TestDetail09_RenderDeterministic(t *testing.T) {
	p := writeHosts(t, []string{
		"127.0.0.1 localhost",
		"",
		"",
		GUARD_BEGIN,
		"10.0.0.2 zebra apple",
		"10.0.0.1 mango",
		"10.0.0.2 cherry",
		GUARD_END,
		"::1 ip6-localhost",
	})
	if err := UpdateHostsFileWithRecords(p, noopMutator); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	out := string(b)
	// Exactly one blank line before begin.
	idx := strings.Index(out, GUARD_BEGIN)
	if idx < 0 {
		t.Fatalf("no begin marker in %q", out)
	}
	before := out[:idx]
	if !strings.HasSuffix(before, "\n\n") || strings.HasSuffix(before, "\n\n\n") {
		t.Fatalf("expected exactly one blank line before block: %q", before[len(before)-6:])
	}
	// Grouped lines: 10.0.0.1 mango; 10.0.0.2 sorted hosts.
	if !strings.Contains(out, "10.0.0.1\tmango") {
		t.Fatalf("missing grouped line 10.0.0.1: %q", out)
	}
	if !strings.Contains(out, "10.0.0.2\tapple cherry zebra") {
		t.Fatalf("missing sorted grouped line 10.0.0.2: %q", out)
	}
	// Address lines sorted: 10.0.0.1 line before 10.0.0.2 line.
	if strings.Index(out, "10.0.0.1\t") > strings.Index(out, "10.0.0.2\t") {
		t.Fatalf("address lines not sorted: %q", out)
	}
	// Unmanaged content preserved.
	if !strings.Contains(out, "127.0.0.1 localhost") || !strings.Contains(out, "::1 ip6-localhost") {
		t.Fatalf("unmanaged content lost: %q", out)
	}
	if !strings.HasSuffix(out, GUARD_END+"\n") && !strings.Contains(out, GUARD_END+"\n") {
		t.Fatalf("end marker missing: %q", out)
	}
}

// Detail 10: byte-identical render -> no write occurs (mtime untouched).
func TestDetail10_NoWriteOnIdentical(t *testing.T) {
	p := writeHosts(t, []string{
		"127.0.0.1 localhost",
		GUARD_BEGIN,
		"10.0.0.1 h1",
		GUARD_END,
	})
	if err := UpdateHostsFileWithRecords(p, noopMutator); err != nil {
		t.Fatal(err)
	}
	fi1, _ := os.Stat(p)
	// Make the file look old so a rewrite would be observable.
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	if err := UpdateHostsFileWithRecords(p, noopMutator); err != nil {
		t.Fatal(err)
	}
	fi2, _ := os.Stat(p)
	if !fi2.ModTime().Equal(old) {
		t.Fatalf("identical render rewrote file: %v -> %v", fi1.ModTime(), fi2.ModTime())
	}
}

// Detail 11: pseudoAtomicWrite — write, pause, re-read, retry; the file
// ends with the requested bytes and mode.
func TestDetail11_PseudoAtomicWriteContent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "atomic")
	payload := []byte("payload-" + strconv.Itoa(int(hiddenSeed())))
	if err := pseudoAtomicWrite(p, payload, 0o640); err != nil {
		t.Fatalf("pseudoAtomicWrite: %v", err)
	}
	got, err := os.ReadFile(p)
	if err != nil || string(got) != string(payload) {
		t.Fatalf("content %q err %v", got, err)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o640 {
		t.Fatalf("mode %v want 640", fi.Mode().Perm())
	}
}

// Detail 12: UpdateHostsFileWithRecords holds a process-wide mutex for the
// whole read-mutate-write — concurrent calls on DIFFERENT files still
// serialize (observable: no interleaved failure; both files correct).
func TestDetail12_UpdateSerializes(t *testing.T) {
	var p1, p2 string
	p1 = writeHosts(t, []string{"127.0.0.1 a", GUARD_BEGIN, "10.0.0.1 h", GUARD_END})
	p2 = writeHosts(t, []string{"127.0.0.1 b", GUARD_BEGIN, "10.0.0.2 h", GUARD_END})
	done := make(chan error, 2)
	go func() { done <- UpdateHostsFileWithRecords(p1, noopMutator) }()
	go func() { done <- UpdateHostsFileWithRecords(p2, noopMutator) }()
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatalf("concurrent update: %v", err)
		}
	}
	for _, p := range []string{p1, p2} {
		out := readHosts(t, p)
		if !strings.Contains(strings.Join(out, "\n"), GUARD_BEGIN) {
			t.Fatalf("file %q corrupted", p)
		}
	}
}
