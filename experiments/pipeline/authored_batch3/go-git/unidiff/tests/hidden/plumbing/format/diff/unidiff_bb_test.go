package diff

import (
	"bytes"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/filemode"
)

// --- stubs ---

type tFile struct {
	hash plumbing.Hash
	mode filemode.FileMode
	path string
}

func (f *tFile) Hash() plumbing.Hash     { return f.hash }
func (f *tFile) Mode() filemode.FileMode { return f.mode }
func (f *tFile) Path() string            { return f.path }

type tChunk struct {
	content string
	op      Operation
}

func (c *tChunk) Content() string { return c.content }
func (c *tChunk) Type() Operation { return c.op }

type tFilePatch struct {
	binary bool
	from   File
	to     File
	chunks []Chunk
}

func (p *tFilePatch) IsBinary() bool      { return p.binary }
func (p *tFilePatch) Files() (File, File) { return p.from, p.to }
func (p *tFilePatch) Chunks() []Chunk     { return p.chunks }

type tPatch struct {
	msg   string
	files []FilePatch
}

func (p *tPatch) Message() string          { return p.msg }
func (p *tPatch) FilePatches() []FilePatch { return p.files }

var (
	udH1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	udH2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
)

func enc(t *testing.T, ctx int, p Patch) string {
	t.Helper()
	var buf bytes.Buffer
	if err := NewUnifiedEncoder(&buf, ctx).Encode(p); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// TestDetail01: `diff --git a/<from> b/<to>`; mode change adds old/new mode
// in octal; rename adds rename from/to; content change adds `index h..h`
// with the mode suffix only when the mode did NOT change.
func TestDetail01(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "a.txt"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "a.txt"}
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{
			&tChunk{content: "old\n", op: Delete},
			&tChunk{content: "new\n", op: Add},
		},
	}}})
	lines := strings.Split(out, "\n")
	if lines[0] != "diff --git a/a.txt b/a.txt" {
		t.Fatalf("first line = %q", lines[0])
	}
	if !strings.Contains(out, "index "+udH1.String()[:7]) &&
		!strings.Contains(out, "index "+udH1.String()) {
		t.Fatalf("no index line: %q", out)
	}
	// Mode suffix only when mode unchanged.
	if !strings.Contains(out, "index "+udH1.String()+".."+udH2.String()+" 100644") &&
		!strings.Contains(out, "index "+udH1.String()[:7]+".."+udH2.String()[:7]+" 100644") {
		t.Fatalf("unchanged-mode index lacks mode suffix: %q", out)
	}
}

// TestDetail02: new file emits `new file mode`, `index 0000..hash`,
// `--- /dev/null` + `+++ b/<to>`; the diff --git line uses the destination
// under both prefixes.
func TestDetail02(t *testing.T) {
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "new.txt"}
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		to:     to,
		chunks: []Chunk{&tChunk{content: "hi\n", op: Add}},
	}}})
	if !strings.Contains(out, "new file mode") {
		t.Fatalf("no new-file mode: %q", out)
	}
	if !strings.Contains(out, "--- /dev/null") || !strings.Contains(out, "+++ b/new.txt") {
		t.Fatalf("missing /dev/null pair: %q", out)
	}
	first := strings.SplitN(out, "\n", 2)[0]
	if first != "diff --git a/new.txt b/new.txt" {
		t.Fatalf("new-file diff line = %q, want dest under both prefixes", first)
	}
}

// TestDetail03 (shape — Inferable: no): a metadata-only change (same hash)
// produces a header with no index line and no hunks.
func TestDetail03(t *testing.T) {
	f := &tFile{hash: udH1, mode: filemode.Regular, path: "a.txt"}
	to := &tFile{hash: udH1, mode: filemode.Executable, path: "a.txt"}
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: f, to: to, chunks: nil,
	}}})
	if strings.Contains(out, "index ") || strings.Contains(out, "@@") ||
		strings.Contains(out, "--- ") || strings.Contains(out, "+++ ") {
		t.Fatalf("metadata-only change emitted body lines: %q", out)
	}
}

// TestDetail04: a binary patch replaces the ---/+++ pair with a single
// `Binary files X and Y differ` line, still after the index line.
func TestDetail04(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "bin.dat"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "bin.dat"}
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to, binary: true,
	}}})
	if strings.Contains(out, "--- ") || strings.Contains(out, "+++ ") {
		t.Fatalf("binary patch emitted ---/+++: %q", out)
	}
	if !strings.Contains(out, "Binary files") || !strings.Contains(out, "differ") {
		t.Fatalf("no binary marker: %q", out)
	}
	if !strings.Contains(out, "index ") {
		t.Fatalf("binary patch missing index line: %q", out)
	}
}

// TestDetail05: hunk header `@@ -l[,c] +l[,c] @@` — `,count` omitted when 1,
// present (`,0`) for an empty side.
func TestDetail05(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{
			&tChunk{content: "ctx1\nctx2\nctx3\n", op: Equal},
			&tChunk{content: "old\n", op: Delete},
			&tChunk{content: "new\n", op: Add},
			&tChunk{content: "ctx4\nctx5\nctx6\n", op: Equal},
		},
	}}})
	if !strings.Contains(out, "@@") {
		t.Fatalf("no hunk: %q", out)
	}
	// One-line changes: `,1` omitted.
	if strings.Contains(out, ",1 ") || strings.Contains(out, ",1 @@") {
		t.Fatalf("count=1 not omitted: %q", out)
	}
}

// TestDetail06 (shape — Inferable: no): the trimmed context line before a
// hunk reappears after `@@` as a heading — committed shape is that a heading
// may follow the second @@ marker.
func TestDetail06(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	// 10 context lines then a change: with context=3 the 7th-back line is
	// trimmed and becomes the section heading.
	var chunks []Chunk
	var ctx strings.Builder
	for i := 0; i < 10; i++ {
		ctx.WriteString("c")
		ctx.WriteString(strings.Repeat("x", 1))
		ctx.WriteByte('\n')
	}
	chunks = append(chunks,
		&tChunk{content: ctx.String(), op: Equal},
		&tChunk{content: "old\n", op: Delete},
		&tChunk{content: "new\n", op: Add},
	)
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{from: from, to: to, chunks: chunks}}})
	var hunkLine string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "@@") {
			hunkLine = l
			break
		}
	}
	if hunkLine == "" {
		t.Fatalf("no hunk header: %q", out)
	}
	// The header line must close its second @@ on the same line; any heading
	// after it is ` <text>` with no embedded newline.
	tail := hunkLine[strings.LastIndex(hunkLine, "@@")+2:]
	if tail != "" && !strings.HasPrefix(tail, " ") {
		t.Fatalf("hunk header tail = %q, want empty or ' <heading>'", tail)
	}
	if strings.Contains(tail, "\n") {
		t.Fatal("heading carried a newline")
	}
}

// TestDetail07: unchanged runs ≤ 2×context keep one hunk; longer runs split.
func TestDetail07(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	// 4 unchanged lines between two changes, context=3 → 4 ≤ 6 → one hunk.
	mid := strings.Repeat("m\n", 4)
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{
			&tChunk{content: "a\n", op: Delete},
			&tChunk{content: mid, op: Equal},
			&tChunk{content: "b\n", op: Delete},
		},
	}}})
	if strings.Count(out, "@@") != 2 { // one hunk = one @@ pair
		t.Fatalf("4-line gap split into multiple hunks: %q", out)
	}
	// 10-line gap → split.
	mid = strings.Repeat("m\n", 10)
	out = enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{
			&tChunk{content: "a\n", op: Delete},
			&tChunk{content: mid, op: Equal},
			&tChunk{content: "b\n", op: Delete},
		},
	}}})
	if strings.Count(out, "@@") != 4 { // two hunks
		t.Fatalf("10-line gap not split: %q", out)
	}
}

// TestDetail08 (shape — Inferable: no): with context 0, adjacent changes
// still merge and no context lines are emitted.
func TestDetail08(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	out := enc(t, 0, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{
			&tChunk{content: "a\n", op: Delete},
			&tChunk{content: "b\n", op: Add},
		},
	}}})
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, " ") && !strings.HasPrefix(l, " @@") {
			t.Fatalf("context line emitted with context=0: %q", l)
		}
	}
}

// TestDetail09: a content line without trailing newline is followed by
// `\ No newline at end of file` on its own line.
func TestDetail09(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{&tChunk{content: "nonewline", op: Delete}},
	}}})
	if !strings.Contains(out, "\\ No newline at end of file") {
		t.Fatalf("missing no-newline marker: %q", out)
	}
}

// TestDetail10: a non-empty patch message precedes all file patches, with a
// newline appended when missing.
func TestDetail10(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	out := enc(t, 3, &tPatch{
		msg:   "the message",
		files: []FilePatch{&tFilePatch{from: from, to: to}},
	})
	if !strings.HasPrefix(out, "the message\n") {
		t.Fatalf("message not prepended with newline: %q", out)
	}
}

// TestDetail11: the line splitter keeps each line's newline; trailing
// newline yields no phantom empty line.
func TestDetail11(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	out := enc(t, 3, &tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{&tChunk{content: "l1\nl2\n", op: Delete}},
	}}})
	body := strings.Split(out, "\n")
	var minus int
	for _, l := range body {
		if strings.HasPrefix(l, "-") && !strings.HasPrefix(l, "---") {
			minus++
		}
	}
	if minus != 2 { // exactly l1,l2 — no phantom third delete line
		t.Fatalf("delete lines = %d, want 2: %q", minus, out)
	}
}

// TestDetail12 (shape — Inferable: no): color spans wrap header, hunk
// header, and per-operation content lines, each reset afterwards.
func TestDetail12(t *testing.T) {
	from := &tFile{hash: udH1, mode: filemode.Regular, path: "f"}
	to := &tFile{hash: udH2, mode: filemode.Regular, path: "f"}
	var buf bytes.Buffer
	e := NewUnifiedEncoder(&buf, 3)
	e.SetColor(ColorConfig{
		New: "\033[32m", Old: "\033[31m", Frag: "\033[36m", Meta: "\033[35m",
	})
	err := e.Encode(&tPatch{files: []FilePatch{&tFilePatch{
		from: from, to: to,
		chunks: []Chunk{
			&tChunk{content: "o\n", op: Delete},
			&tChunk{content: "n\n", op: Add},
		},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, key := range []string{"\033[32m", "\033[31m", "\033[36m"} {
		if !strings.Contains(out, key) {
			t.Fatalf("color %q missing: %q", key, out)
		}
	}
	if !strings.Contains(out, "\033[32m+n") {
		t.Fatalf("added line not wrapped in New color: %q", out)
	}
}
