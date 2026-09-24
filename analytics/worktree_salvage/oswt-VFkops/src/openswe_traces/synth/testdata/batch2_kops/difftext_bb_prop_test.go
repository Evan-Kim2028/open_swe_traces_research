package diff_test

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"example.internal/kops/pkg/diff"
)

func hiddenSeed() int64 {
	if s := os.Getenv("HIDDEN_SEED"); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n
		}
	}
	return 20260919
}

func mkLines(rng *rand.Rand, n int, tag string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s-line-%d-%d", tag, i, rng.Intn(1000))
	}
	return out
}

// classify output lines: "..." elision, "- x" deletion, "+ x" insertion, "  x" context.
func parseOut(t *testing.T, s string) [][]string {
	if !strings.HasSuffix(s, "\n") && s != "" {
		t.Fatalf("output does not end with newline: %q", s[len(s)-8:])
	}
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	var recs [][]string
	for _, l := range lines {
		switch {
		case l == "...":
			recs = append(recs, []string{"e"})
		case strings.HasPrefix(l, "- "):
			recs = append(recs, []string{"-", l[2:]})
		case strings.HasPrefix(l, "+ "):
			recs = append(recs, []string{"+", l[2:]})
		case strings.HasPrefix(l, "  "):
			recs = append(recs, []string{" ", l[2:]})
		default:
			t.Fatalf("unprefixed output line %q", l)
		}
	}
	return recs
}

func opsOf(recs [][]string) string {
	var b strings.Builder
	for _, r := range recs {
		b.WriteString(r[0])
	}
	return b.String()
}

// Detail 1: a changed line yields exactly one insertion and one deletion
// record; unchanged surrounding lines appear as context/elision.
func TestDetail01_ChangedLineYieldsOneInsertOneDelete(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 300; i++ {
		n := 3 + rng.Intn(10)
		l := mkLines(rng, n, "a")
		r := append([]string{}, l...)
		pos := rng.Intn(n)
		r[pos] = fmt.Sprintf("changed-%d-%d", pos, i)
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		recs := parseOut(t, out)
		var ins, del int
		for _, rec := range recs {
			if rec[0] == "+" {
				ins++
				if rec[1] != r[pos] {
					t.Fatalf("insert line %q want %q", rec[1], r[pos])
				}
			}
			if rec[0] == "-" {
				del++
				if rec[1] != l[pos] {
					t.Fatalf("delete line %q want %q", rec[1], l[pos])
				}
			}
		}
		if ins != 1 || del != 1 {
			t.Fatalf("i=%d got %d ins %d del records for single changed line: %q", i, ins, del, out)
		}
	}
}

// Detail 2: mid-document changed line emits insertion BEFORE deletion.
func TestDetail02_MidDocInsertBeforeDelete(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 300; i++ {
		n := 6 + rng.Intn(10)
		l := mkLines(rng, n, "a")
		r := append([]string{}, l...)
		pos := 2 + rng.Intn(n-4) // strictly mid-document with lines after
		r[pos] = fmt.Sprintf("changed-%d-%d", pos, i)
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		ops := opsOf(parseOut(t, out))
		pi := strings.Index(ops, "+")
		di := strings.Index(ops, "-")
		if pi < 0 || di < 0 {
			t.Fatalf("i=%d missing change records in %q", i, out)
		}
		if pi > di {
			t.Fatalf("i=%d mid-doc delete before insert in %q (ops %q)", i, out, ops)
		}
	}
}

// Detail 3: when the changed last line reaches the very end of the input
// (no trailing newline), deletion is emitted BEFORE insertion — the
// opposite order of commitment 2.
func TestDetail03_EndOfInputDeleteBeforeInsert(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 300; i++ {
		n := 2 + rng.Intn(10)
		l := mkLines(rng, n, "a")
		r := append([]string{}, l...)
		r[n-1] = fmt.Sprintf("changed-last-%d", i)
		// No trailing newline: the change reaches end-of-input.
		out := diff.FormatDiff(strings.Join(l, "\n"), strings.Join(r, "\n"))
		ops := opsOf(parseOut(t, out))
		pi := strings.Index(ops, "+")
		di := strings.Index(ops, "-")
		if pi < 0 || di < 0 {
			t.Fatalf("i=%d missing change records in %q", i, out)
		}
		if di > pi {
			t.Fatalf("i=%d end-of-input insert before delete in %q (ops %q)", i, out, ops)
		}
	}
	// Sanity: same change with trailing newline keeps insert-first order.
	l := []string{"x1", "x2", "x3"}
	r := []string{"x1", "x2", "CHANGED"}
	out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
	ops := opsOf(parseOut(t, out))
	if strings.Index(ops, "+") > strings.Index(ops, "-") {
		t.Fatalf("trailing-newline case flipped: %q", out)
	}
}

// Detail 4: context window is exactly 2 records on each side of a change;
// windows separated by <=4 unchanged records merge, wider gaps elide.
func TestDetail04_ContextWindowTwoWithMerge(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 400; i++ {
		gap := 1 + rng.Intn(10)
		n := 4 + gap + 4
		l := mkLines(rng, n, "a")
		r := append([]string{}, l...)
		p1 := 2 + rng.Intn(2)
		p2 := p1 + 2 + gap + 2
		if p2 >= n {
			continue
		}
		r[p1] = fmt.Sprintf("chg1-%d", i)
		r[p2] = fmt.Sprintf("chg2-%d", i)
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		recs := parseOut(t, out)
		// Locate the elision records strictly between the two change groups.
		var changeIdx []int
		for j, rec := range recs {
			if rec[0] == "+" || rec[0] == "-" {
				changeIdx = append(changeIdx, j)
			}
		}
		if len(changeIdx) < 4 {
			t.Fatalf("i=%d expected two changes in %q", i, out)
		}
		midElide := 0
		for j := changeIdx[1] + 1; j < changeIdx[len(changeIdx)-2]; j++ {
			if recs[j][0] == "e" {
				midElide++
			}
		}
		// Middle equal-run: windows of 2 cover up to 4 records; a longer
		// run leaves a skipped middle rendered as one elision line.
		middle := p2 - p1 - 1
		if middle > 4 && midElide == 0 {
			t.Fatalf("i=%d middle=%d expected elision in %q", i, middle, out)
		}
		if middle <= 4 && midElide != 0 {
			t.Fatalf("i=%d middle=%d unexpected elision in %q", i, middle, out)
		}
	}
	// Focused: single change far into a large doc keeps exactly 2 context
	// records before and after.
	for i := 0; i < 100; i++ {
		n := 15 + rng.Intn(10)
		l := mkLines(rng, n, "b")
		r := append([]string{}, l...)
		pos := 6 + rng.Intn(n-12)
		r[pos] = fmt.Sprintf("solo-%d", i)
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		recs := parseOut(t, out)
		var before, after int
		seenChange := false
		for _, rec := range recs {
			switch rec[0] {
			case " ":
				if seenChange {
					after++
				} else {
					before++
				}
			case "+", "-":
				seenChange = true
			}
		}
		if before != 2 || after != 2 {
			t.Fatalf("i=%d context before=%d after=%d want 2/2 in %q", i, before, after, out)
		}
	}
}

// Detail 5: every contiguous skipped run renders as exactly one "...\n",
// including runs at the very start and end of output.
func TestDetail05_SkippedRunIsOneElisionLine(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		n := 12 + rng.Intn(10)
		l := mkLines(rng, n, "a")
		r := append([]string{}, l...)
		r[5] = fmt.Sprintf("mid-%d", i)
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		if strings.Count(out, "...\n") != 2 {
			t.Fatalf("i=%d expected exactly 2 elision lines (leading+trailing) in %q", i, out)
		}
		if strings.Contains(out, "...\n...") {
			t.Fatalf("i=%d consecutive elisions in %q", i, out)
		}
		// No more than one "..." per skipped region: "......" never appears.
		if strings.Contains(out, "....") {
			t.Fatalf("i=%d doubled dots in %q", i, out)
		}
	}
}

// Detail 6: line prefixes are exactly "- ", "+ ", "  " and every emitted
// line ends in "\n".
func TestDetail06_LinePrefixesAndTrailingNewline(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 300; i++ {
		n := 2 + rng.Intn(15)
		l := mkLines(rng, n, "a")
		r := append([]string{}, l...)
		// Sprinkle random edits: substitutions, inserts, deletes.
		for j := 0; j < n; j++ {
			switch rng.Intn(8) {
			case 0:
				r[j] = fmt.Sprintf("edit-%d-%d", i, j)
			case 1:
				r[j] = r[j] + "-suffix"
			}
		}
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		if out == "" {
			continue
		}
		if !strings.HasSuffix(out, "\n") {
			t.Fatalf("i=%d output missing trailing newline: %q", i, out[len(out)-16:])
		}
		if strings.Contains(out, "\n\n") {
			t.Fatalf("i=%d blank line in output %q", i, out)
		}
		parseOut(t, out) // panics on any unprefixed line
	}
}

// Detail 7: trailing-newline asymmetry counts as a changed last line —
// "- x"/"+ x" pair of identical text.
func TestDetail07_TrailingNewlineAsymmetry(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		n := 1 + rng.Intn(8)
		l := mkLines(rng, n, "a")
		withNL := strings.Join(l, "\n") + "\n"
		withoutNL := strings.Join(l, "\n")
		out := diff.FormatDiff(withNL, withoutNL)
		recs := parseOut(t, out)
		var del, ins string
		for _, rec := range recs {
			if rec[0] == "-" {
				del = rec[1]
			}
			if rec[0] == "+" {
				ins = rec[1]
			}
		}
		if del != l[n-1] || ins != l[n-1] {
			t.Fatalf("i=%d trailing-newline diff = -%q/+%q want identical %q", i, del, ins, l[n-1])
		}
	}
}

// Detail 8: identical inputs render as a single "...\n" — not empty output.
func TestDetail08_IdenticalInputsRenderSingleElision(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 100; i++ {
		n := 1 + rng.Intn(20)
		l := mkLines(rng, n, "a")
		s := strings.Join(l, "\n")
		if rng.Intn(2) == 0 {
			s += "\n"
		}
		out := diff.FormatDiff(s, s)
		if out != "...\n" {
			t.Fatalf("i=%d identical input rendered %q want \"...\\n\"", i, out)
		}
	}
}

// Detail 9: partial-line edits coalesce to whole-line records — no
// intra-line fragments in output.
func TestDetail09_PartialLineEditsCoalesceToWholeLines(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 300; i++ {
		n := 4 + rng.Intn(8)
		l := mkLines(rng, n, "a")
		r := append([]string{}, l...)
		pos := rng.Intn(n)
		// Change only a fragment of the line.
		r[pos] = l[pos] + "-TAIL" + strconv.Itoa(i)
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		recs := parseOut(t, out)
		for _, rec := range recs {
			if rec[0] == "-" && rec[1] != l[pos] {
				t.Fatalf("i=%d delete record is a fragment %q want whole %q", i, rec[1], l[pos])
			}
			if rec[0] == "+" && rec[1] != r[pos] {
				t.Fatalf("i=%d insert record is a fragment %q want whole %q", i, rec[1], r[pos])
			}
		}
	}
}

// Detail 10: pending partial-line fragments accumulate across consecutive
// insert/delete runs until a newline boundary — edits that join or split
// lines still produce whole-line records.
func TestDetail10_FragmentsAccumulateAcrossRuns(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		n := 5 + rng.Intn(6)
		l := mkLines(rng, n, "a")
		// Merge two adjacent lines by deleting the newline between them,
		// and also tweak the merged content.
		pos := rng.Intn(n - 1)
		merged := l[pos] + "-JOINED-" + l[pos+1]
		r := append(append([]string{}, l[:pos]...), append([]string{merged}, l[pos+2:]...)...)
		out := diff.FormatDiff(strings.Join(l, "\n")+"\n", strings.Join(r, "\n")+"\n")
		recs := parseOut(t, out)
		sawWhole := false
		for _, rec := range recs {
			if rec[0] != "+" && rec[0] != "-" {
				continue
			}
			if rec[0] == "+" && rec[1] == merged {
				sawWhole = true
			}
			// record text must be a complete input/output line, never a
			// fragment spanning or splitting a newline boundary
			ok := false
			for _, ll := range l {
				if rec[1] == ll {
					ok = true
				}
			}
			for _, rr := range r {
				if rec[1] == rr {
					ok = true
				}
			}
			if !ok {
				t.Fatalf("i=%d fragment record %q in %q", i, rec[1], out)
			}
		}
		if !sawWhole {
			t.Fatalf("i=%d merged line %q absent from %q", i, merged, out)
		}
	}
}

// Detail 11: wholly-deleted or wholly-inserted documents render only
// "- "/"+ " lines, no elision.
func TestDetail11_WholeDocChangesRenderNoElision(t *testing.T) {
	rng := rand.New(rand.NewSource(hiddenSeed()))
	for i := 0; i < 200; i++ {
		n := 1 + rng.Intn(12)
		l := mkLines(rng, n, "a")
		doc := strings.Join(l, "\n") + "\n"
		out := diff.FormatDiff(doc, "")
		recs := parseOut(t, out)
		if len(recs) != n {
			t.Fatalf("i=%d delete-all rendered %d records want %d: %q", i, len(recs), n, out)
		}
		for j, rec := range recs {
			if rec[0] != "-" || rec[1] != l[j] {
				t.Fatalf("i=%d delete-all record %v want - %q", i, rec, l[j])
			}
		}
		out = diff.FormatDiff("", doc)
		recs = parseOut(t, out)
		if len(recs) != n {
			t.Fatalf("i=%d insert-all rendered %d records want %d: %q", i, len(recs), n, out)
		}
		for j, rec := range recs {
			if rec[0] != "+" || rec[1] != l[j] {
				t.Fatalf("i=%d insert-all record %v want + %q", i, rec, l[j])
			}
		}
	}
}
