package packp

import (
	"bytes"
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

var (
	fh1, _ = plumbing.FromHex("1111111111111111111111111111111111111111")
	fh2, _ = plumbing.FromHex("2222222222222222222222222222222222222222")
	fh3, _ = plumbing.FromHex("3333333333333333333333333333333333333333")
	fhA, _ = plumbing.FromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	fhB, _ = plumbing.FromHex("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
)

// frames builds a wire stream: each string is one pkt-line payload; the
// tokens <flush> and <delim> emit the special packets.
func frames(parts ...string) *bytes.Buffer {
	var buf bytes.Buffer
	for _, p := range parts {
		switch p {
		case "<flush>":
			if err := pktline.WriteFlush(&buf); err != nil {
				panic(err)
			}
		case "<delim>":
			if err := pktline.WriteDelim(&buf); err != nil {
				panic(err)
			}
		default:
			if _, err := pktline.WriteString(&buf, p); err != nil {
				panic(err)
			}
		}
	}
	return &buf
}

// frame is one scanned pkt-line: kind 0/1/2 for flush/delim/response-end,
// -1 for a data line whose payload is text.
type frame struct {
	text string
	kind int
}

func scanFrames(t *testing.T, b []byte) []frame {
	t.Helper()
	var out []frame
	s := pktline.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		f := frame{text: s.Text(), kind: -1}
		switch s.Len() {
		case pktline.Flush:
			f.kind = 0
		case pktline.Delim:
			f.kind = 1
		case pktline.ResponseEnd:
			f.kind = 2
		}
		out = append(out, f)
	}
	if err := s.Err(); err != nil {
		t.Fatalf("scanner: %v", err)
	}
	return out
}

func dataLines(fs []frame) []string {
	var out []string
	for _, f := range fs {
		if f.kind == -1 {
			out = append(out, f.text)
		}
	}
	return out
}

func decodeArgsWithin(t *testing.T, a *FetchArgs, buf *bytes.Buffer) (err error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- a.Decode(buf) }()
	select {
	case err = <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("FetchArgs.Decode did not return within 5s")
		return nil
	}
}

func decodeOutputWithin(t *testing.T, o *FetchOutput, buf *bytes.Buffer) (err error) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- o.Decode(buf) }()
	select {
	case err = <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("FetchOutput.Decode did not return within 5s")
		return nil
	}
}

func isMalformedErr(err error) bool {
	var mre *MalformedResponseError
	return errors.As(err, &mre)
}

// TestDetail01 (shape — Inferable: no): want, have and shallow lines are each
// emitted sorted by hash; the caller's slice order is ignored.
func TestDetail01(t *testing.T) {
	args := &FetchArgs{
		Wants:    []plumbing.Hash{fh3, fh1, fh2},
		Haves:    []plumbing.Hash{fhB, fhA},
		Shallows: []plumbing.Hash{fh2, fh1},
	}
	var buf bytes.Buffer
	if err := args.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	lines := dataLines(scanFrames(t, buf.Bytes()))

	byKey := map[string][]string{}
	for _, l := range lines {
		f := strings.Fields(l)
		if len(f) == 2 {
			byKey[f[0]] = append(byKey[f[0]], f[1])
		}
	}
	for _, key := range []string{"want", "have", "shallow"} {
		got := byKey[key]
		if len(got) == 0 {
			t.Fatalf("no %q lines emitted: %v", key, lines)
		}
		if !sort.StringsAreSorted(got) {
			t.Fatalf("%s lines not sorted by hash: %v", key, got)
		}
	}
	if got := byKey["want"]; len(got) != 3 || got[0] != fh1.String() || got[2] != fh3.String() {
		t.Fatalf("want lines = %v, want ascending 1111.., 2222.., 3333..", got)
	}
}

// TestDetail02 (shape — Inferable: no): Encode with an empty want list errors
// before writing anything.
func TestDetail02(t *testing.T) {
	var buf bytes.Buffer
	args := &FetchArgs{Haves: []plumbing.Hash{fh1}, Done: true}
	if err := args.Encode(&buf); err == nil {
		t.Fatal("Encode with empty Wants returned nil error")
	}
	if buf.Len() != 0 {
		t.Fatalf("Encode wrote %d bytes before failing on empty wants", buf.Len())
	}
}

// TestDetail03: every set flag emits its own line; encoding is deterministic —
// the same arguments never reorder by value.
func TestDetail03(t *testing.T) {
	args := &FetchArgs{
		Wants:          []plumbing.Hash{fh1},
		Done:           true,
		ThinPack:       true,
		NoProgress:     true,
		IncludeTag:     true,
		OFSDelta:       true,
		Shallows:       []plumbing.Hash{fh2},
		Deepen:         4,
		DeepenRelative: true,
		DeepenSince:    time.Unix(1700000000, 0),
		DeepenNot:      []string{"refs/heads/skip"},
		Filter:         FilterBlobNone(),
		WaitForDone:    true,
	}
	var buf bytes.Buffer
	if err := args.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	lines := dataLines(scanFrames(t, buf.Bytes()))
	joined := strings.Join(lines, "")

	for _, want := range []string{
		"done", "thin-pack", "no-progress", "include-tag", "ofs-delta",
		"deepen ", "deepen-relative", "deepen-since ", "deepen-not ",
		"filter ", "wait-for-done", "shallow ",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("encoded args missing %q: %v", want, lines)
		}
	}

	var buf2 bytes.Buffer
	if err := args.Encode(&buf2); err != nil {
		t.Fatalf("Encode (2): %v", err)
	}
	if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
		t.Fatal("Encode is not deterministic for identical input")
	}
}

// TestDetail04: a set DeepenRelative emits a deepen-relative line; the depth
// itself is carried by the deepen line.
func TestDetail04(t *testing.T) {
	args := &FetchArgs{Wants: []plumbing.Hash{fh1}, Deepen: 7, DeepenRelative: true}
	var buf bytes.Buffer
	if err := args.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	lines := dataLines(scanFrames(t, buf.Bytes()))

	var sawRel, sawDeep bool
	for _, l := range lines {
		f := strings.Fields(l)
		if f[0] == "deepen-relative" {
			sawRel = true
			if len(f) != 1 {
				t.Fatalf("deepen-relative emitted with an argument: %q", l)
			}
		}
		if f[0] == "deepen" {
			sawDeep = true
			if len(f) != 2 || f[1] != "7" {
				t.Fatalf("deepen line = %q, want \"deepen 7\"", l)
			}
		}
	}
	if !sawRel || !sawDeep {
		t.Fatalf("want deepen + deepen-relative lines: %v", lines)
	}
}

// TestDetail05: deepen-since encodes as UTC unix seconds.
func TestDetail05(t *testing.T) {
	loc := time.FixedZone("UTC+5", 5*3600)
	ts := time.Date(2024, 1, 1, 12, 0, 0, 0, loc)
	args := &FetchArgs{Wants: []plumbing.Hash{fh1}, DeepenSince: ts}
	var buf bytes.Buffer
	if err := args.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var found string
	for _, l := range dataLines(scanFrames(t, buf.Bytes())) {
		if strings.HasPrefix(l, "deepen-since") {
			found = l
		}
	}
	want := "deepen-since " + strconv.FormatInt(ts.Unix(), 10)
	if strings.TrimSuffix(found, "\n") != want {
		t.Fatalf("deepen-since line = %q, want %q (unix seconds of the instant)", found, want)
	}
}

// TestDetail06: decode ends without error on flush-pkt, delim-pkt,
// end-of-input, or a blank line.
func TestDetail06(t *testing.T) {
	for name, tail := range map[string][]string{
		"flush": {"<flush>"},
		"delim": {"<delim>"},
		"eof":   {},
		"blank": {"\n"},
	} {
		var a FetchArgs
		parts := append([]string{"want " + fh1.String() + "\n"}, tail...)
		err := decodeArgsWithin(t, &a, frames(parts...))
		if err != nil {
			t.Fatalf("Decode (%s terminator): %v", name, err)
		}
		if len(a.Wants) != 1 || a.Wants[0] != fh1 {
			t.Fatalf("Decode (%s): Wants = %v", name, a.Wants)
		}
	}
}

// TestDetail07 (shape — Inferable: no): unrecognized lines are skipped, not
// rejected.
func TestDetail07(t *testing.T) {
	var a FetchArgs
	err := decodeArgsWithin(t, &a, frames(
		"want "+fh1.String()+"\n",
		"bogus-flag some args\n",
		"have "+fh2.String()+"\n",
		"<flush>",
	))
	if err != nil {
		t.Fatalf("Decode with unknown line: %v", err)
	}
	if len(a.Wants) != 1 || len(a.Haves) != 1 {
		t.Fatalf("known lines not parsed around unknown line: %+v", a)
	}
}

// TestDetail08 (shape — Inferable: no): a deepen-relative line carrying an
// argument is accepted; only the flag is set.
func TestDetail08(t *testing.T) {
	var a FetchArgs
	err := decodeArgsWithin(t, &a, frames(
		"want "+fh1.String()+"\n",
		"deepen 5\n",
		"deepen-relative 99\n",
		"<flush>",
	))
	if err != nil {
		t.Fatalf("Decode deepen-relative with arg: %v", err)
	}
	if !a.DeepenRelative {
		t.Fatal("DeepenRelative not set")
	}
	if a.Deepen != 5 {
		t.Fatalf("Deepen = %d, want 5 (the arg must be ignored)", a.Deepen)
	}
}

// TestDetail09: hash fields are validated as full-length object IDs.
func TestDetail09(t *testing.T) {
	for _, line := range []string{
		"want abc123\n",
		"want zz" + fh1.String()[2:] + "\n",
	} {
		var a FetchArgs
		if err := decodeArgsWithin(t, &a, frames(line, "<flush>")); err == nil {
			t.Fatalf("Decode(%q): want error", line)
		}
	}
	var a FetchArgs
	if err := decodeArgsWithin(t, &a, frames(
		"want "+fh1.String()+"\n", "have "+fh2.String()+"\n",
		"shallow "+fh3.String()+"\n", "<flush>",
	)); err != nil {
		t.Fatalf("Decode full-length hashes: %v", err)
	}
	if a.Wants[0] != fh1 || a.Haves[0] != fh2 || a.Shallows[0] != fh3 {
		t.Fatalf("hash fields misdecoded: %+v", a)
	}
}

// TestDetail10: each list-like section is capped by maxSectionLines.
func TestDetail10(t *testing.T) {
	defer func() { maxSectionLines = 1 << 22 }()
	maxSectionLines = 2

	var a FetchArgs
	err := decodeArgsWithin(t, &a, frames(
		"want "+fh1.String()+"\n",
		"want "+fh2.String()+"\n",
		"want "+fh3.String()+"\n",
		"<flush>",
	))
	if err == nil {
		t.Fatal("Decode past maxSectionLines: want error")
	}
}

// TestDetail11: response sections are order-enforced — a repeated or
// out-of-order section header is a malformed-response error.
func TestDetail11(t *testing.T) {
	o := &FetchOutput{}
	err := decodeOutputWithin(t, o, frames(
		"shallow-info\n", "<delim>",
		"acknowledgments\n", "NAK\n", "<delim>",
		"packfile\n",
	))
	if !isMalformedErr(err) {
		t.Fatalf("out-of-order sections: err = %v, want MalformedResponseError", err)
	}

	o = &FetchOutput{}
	err = decodeOutputWithin(t, o, frames(
		"acknowledgments\n", "NAK\n", "ready\n", "<delim>",
		"acknowledgments\n", "NAK\n", "<delim>",
		"packfile\n",
	))
	if !isMalformedErr(err) {
		t.Fatalf("repeated section: err = %v, want MalformedResponseError", err)
	}
}

// TestDetail12: a ready line commits the response to the packfile shape — it
// must be followed by a delim-pkt; without ready the acknowledgments section
// must end the whole response.
func TestDetail12(t *testing.T) {
	o := &FetchOutput{}
	err := decodeOutputWithin(t, o, frames(
		"acknowledgments\n", "ACK "+fh1.String()+"\n", "ready\n", "<flush>",
	))
	if !isMalformedErr(err) {
		t.Fatalf("ready followed by flush: err = %v, want malformed", err)
	}

	o = &FetchOutput{}
	err = decodeOutputWithin(t, o, frames(
		"acknowledgments\n", "ready\n",
	))
	if err == nil {
		t.Fatal("ready followed by EOF: want an error, got clean decode")
	}

	o = &FetchOutput{}
	err = decodeOutputWithin(t, o, frames(
		"acknowledgments\n", "NAK\n", "<delim>", "packfile\n",
	))
	if !isMalformedErr(err) {
		t.Fatalf("no-ready acknowledgments continuing: err = %v, want malformed", err)
	}

	o = &FetchOutput{}
	if err := decodeOutputWithin(t, o, frames(
		"acknowledgments\n", "NAK\n", "<flush>",
	)); err != nil {
		t.Fatalf("negotiation round decode: %v", err)
	}
	if o.Packfile {
		t.Fatal("Packfile set on a flush-terminated negotiation round")
	}
}

// TestDetail13: reaching the packfile header sets Packfile and returns with
// the reader positioned at the first packfile pkt-line.
func TestDetail13(t *testing.T) {
	pack := frames(
		"acknowledgments\n", "ready\n", "<delim>",
		"packfile\n",
		"\x01PACKDATA",
		"<flush>",
	)
	o := &FetchOutput{}
	if err := decodeOutputWithin(t, o, pack); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !o.Packfile {
		t.Fatal("Packfile flag not set after packfile header")
	}
	buf := make([]byte, 64)
	n, err := pktline.Read(pack, buf)
	if err != nil {
		t.Fatalf("reading first packfile pkt-line: %v", err)
	}
	if string(buf[pktline.LenSize:n]) != "\x01PACKDATA" {
		t.Fatalf("reader not positioned at first packfile pkt-line: %q", buf[pktline.LenSize:n])
	}
}

// TestDetail14: acknowledgments encode ACK lines first, then a single ready;
// NAK only when there were no ACKs and no ready.
func TestDetail14(t *testing.T) {
	o := &FetchOutput{
		Packfile:        true,
		Acknowledgments: &Acknowledgments{ACKs: []plumbing.Hash{fh1, fh2}, Ready: true},
	}
	var buf bytes.Buffer
	if err := o.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	fs := scanFrames(t, buf.Bytes())
	var texts []string
	for _, f := range fs {
		if f.kind == -1 {
			texts = append(texts, f.text)
		}
	}
	joined := strings.Join(texts, "")
	ackIdx := strings.Index(joined, "ACK "+fh1.String()+"\n")
	readyIdx := strings.Index(joined, "ready\n")
	if ackIdx < 0 || readyIdx < 0 || ackIdx > readyIdx {
		t.Fatalf("ACK lines must precede ready: %v", texts)
	}
	if strings.Count(joined, "ready\n") != 1 {
		t.Fatalf("want exactly one ready line: %v", texts)
	}
	if strings.Contains(joined, "NAK") {
		t.Fatalf("NAK emitted alongside ACKs/ready: %v", texts)
	}

	o = &FetchOutput{Packfile: false, Acknowledgments: &Acknowledgments{}}
	buf.Reset()
	if err := o.Encode(&buf); err != nil {
		t.Fatalf("Encode negotiation round: %v", err)
	}
	texts = dataLines(scanFrames(t, buf.Bytes()))
	joined = strings.Join(texts, "")
	if !strings.Contains(joined, "NAK") {
		t.Fatalf("empty acknowledgments emitted no NAK: %v", texts)
	}
}

// TestDetail15: a no-packfile encode writes acknowledgments + flush-pkt and
// rejects missing acknowledgments, ready acknowledgments, or extra sections.
func TestDetail15(t *testing.T) {
	o := &FetchOutput{Packfile: false, Acknowledgments: &Acknowledgments{ACKs: []plumbing.Hash{fh1}}}
	var buf bytes.Buffer
	if err := o.Encode(&buf); err != nil {
		t.Fatalf("Encode negotiation round: %v", err)
	}
	fs := scanFrames(t, buf.Bytes())
	if n := len(fs); n == 0 || fs[n-1].kind != 0 {
		t.Fatalf("negotiation round not terminated by flush-pkt: %+v", fs)
	}

	for name, o := range map[string]*FetchOutput{
		"no-acks":     {Packfile: false},
		"ready-acks":  {Packfile: false, Acknowledgments: &Acknowledgments{Ready: true}},
		"extra-sect":  {Packfile: false, Acknowledgments: &Acknowledgments{}, ShallowInfo: &ShallowInfo{}},
	} {
		var b bytes.Buffer
		if err := o.Encode(&b); err == nil {
			t.Fatalf("Encode no-packfile %s: want rejection", name)
		}
	}
}

// TestDetail16 (shape — Inferable: no): packfile-uris decode trims only the
// trailing newline; other whitespace inside a URI line is preserved.
func TestDetail16(t *testing.T) {
	uri := "https://example.com/a b.git "
	o := &FetchOutput{}
	err := decodeOutputWithin(t, o, frames(
		"packfile-uris\n", uri+"\n", "<delim>", "packfile\n",
	))
	if err != nil {
		t.Fatalf("Decode packfile-uris: %v", err)
	}
	if o.PackfileURIs == nil || len(o.PackfileURIs.URIs) != 1 {
		t.Fatalf("PackfileURIs = %+v", o.PackfileURIs)
	}
	if o.PackfileURIs.URIs[0] != uri {
		t.Fatalf("URI whitespace trimmed: %q, want %q", o.PackfileURIs.URIs[0], uri)
	}
}
