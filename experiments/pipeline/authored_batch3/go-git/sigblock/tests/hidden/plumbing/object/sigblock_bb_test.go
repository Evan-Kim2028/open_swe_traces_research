package object

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/storage/memory"
)

func sbObj(t *testing.T, typ plumbing.ObjectType, raw string) plumbing.EncodedObject {
	t.Helper()
	obj := memory.NewStorage().NewEncodedObject()
	obj.SetType(typ)
	obj.SetSize(int64(len(raw)))
	w, err := obj.Writer()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return obj
}

func sbRead(t *testing.T, obj plumbing.EncodedObject) string {
	t.Helper()
	r, err := obj.Reader()
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestDetail01: a signature marker is recognised only at a line boundary —
// a marker mid-line does not count.
func TestDetail01(t *testing.T) {
	mid := "some prose -----BEGIN PGP SIGNATURE-----\n"
	if pos, _ := parseSignedBytes([]byte(mid)); pos != -1 {
		t.Fatalf("mid-line marker at pos %d", pos)
	}
	line := "some prose\n-----BEGIN PGP SIGNATURE-----\nsig\n"
	pos, _ := parseSignedBytes([]byte(line))
	if pos != len("some prose\n") {
		t.Fatalf("line-boundary marker pos = %d", pos)
	}
}

// TestDetail02: `-----BEGIN PGP MESSAGE-----` counts as signature material.
func TestDetail02(t *testing.T) {
	b := "msg\n-----BEGIN PGP MESSAGE-----\ndata\n"
	pos, typ := parseSignedBytes([]byte(b))
	if pos != 4 {
		t.Fatalf("PGP MESSAGE marker pos = %d", pos)
	}
	if typ != signatureTypeOpenPGP {
		t.Fatalf("PGP MESSAGE type = %v, want OpenPGP", typ)
	}
}

// TestDetail03: the split position is the LAST block's start.
func TestDetail03(t *testing.T) {
	b := "a\n-----BEGIN PGP SIGNATURE-----\nx\nmore\n-----BEGIN SSH SIGNATURE-----\ny\n"
	pos, typ := parseSignedBytes([]byte(b))
	second := strings.Index(b, "-----BEGIN SSH SIGNATURE-----")
	if pos != second {
		t.Fatalf("pos = %d, want last block at %d", pos, second)
	}
	if typ != signatureTypeSSH {
		t.Fatalf("last block type = %v, want SSH", typ)
	}
}

// TestDetail04: more than one signature-start line counts as
// multi-signature.
func TestDetail04(t *testing.T) {
	b := "m\n-----BEGIN PGP SIGNATURE-----\nx\n-----BEGIN PGP SIGNATURE-----\ny\n"
	if n := countSignatureBlocks([]byte(b)); n < 2 {
		t.Fatalf("countSignatureBlocks = %d, want >= 2", n)
	}
	if n := countSignatureBlocks([]byte("m\n-----BEGIN PGP SIGNATURE-----\nx\n")); n != 1 {
		t.Fatalf("single block = %d", n)
	}
}

// TestDetail05: signature headers match only `gpgsig ` and `gpgsig-sha256 `
// with trailing space.
func TestDetail05(t *testing.T) {
	for _, yes := range []string{"gpgsig ", "gpgsig-sha256 "} {
		if !isSignatureHeader([]byte(yes)) {
			t.Fatalf("isSignatureHeader(%q) = false", yes)
		}
	}
	for _, no := range []string{"gpgsig", "gpgsigfoo", "gpgsig-x ", "gpgsig2 "} {
		if isSignatureHeader([]byte(no)) {
			t.Fatalf("isSignatureHeader(%q) = true", no)
		}
	}
}

// TestDetail06 (shape — Inferable: no): stripping drops each signature
// header and its space-continuation lines; a second signature header right
// after is also dropped; other headers and the body survive.
func TestDetail06(t *testing.T) {
	in := "object x\ngpgsig sig\n cont-line\ngpgsig-sha256 s2\n cont2\nother keep\n\nbody line\n"
	var buf bytes.Buffer
	if err := stripHeaderSignatures(&buf, strings.NewReader(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "gpgsig") || strings.Contains(out, "cont-line") ||
		strings.Contains(out, "cont2") {
		t.Fatalf("signature material survived: %q", out)
	}
	if !strings.Contains(out, "object x") || !strings.Contains(out, "other keep") ||
		!strings.Contains(out, "body line") {
		t.Fatalf("non-signature content lost: %q", out)
	}
}

// TestDetail07: after the blank line closing the header block, everything is
// copied verbatim — a gpgsig-looking body line is never stripped.
func TestDetail07(t *testing.T) {
	in := "object x\ngpgsig sig\n\nbody\ngpgsig bodyline\n"
	var buf bytes.Buffer
	if err := stripHeaderSignatures(&buf, strings.NewReader(in)); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "gpgsig bodyline") {
		t.Fatalf("body gpgsig line stripped: %q", out)
	}
	if strings.Contains(out, "gpgsig sig") {
		t.Fatalf("header gpgsig kept: %q", out)
	}
}

// TestDetail08: tag objects get their trailing inline signature truncated;
// commits do not.
func TestDetail08(t *testing.T) {
	raw := "hdr v\n\nmsg\n-----BEGIN PGP SIGNATURE-----\nINLINE\n"
	srcT := sbObj(t, plumbing.TagObject, raw)
	dstT := memory.NewStorage().NewEncodedObject()
	if err := stripObjectSignatures(dstT, srcT, plumbing.TagObject); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sbRead(t, dstT), "INLINE") {
		t.Fatal("tag inline signature not truncated")
	}

	srcC := sbObj(t, plumbing.CommitObject, raw)
	dstC := memory.NewStorage().NewEncodedObject()
	if err := stripObjectSignatures(dstC, srcC, plumbing.CommitObject); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sbRead(t, dstC), "INLINE") {
		t.Fatal("commit body signature truncated")
	}
}

// TestDetail09 (shape — Inferable: no): a final line with no trailing
// newline is still processed and copied.
func TestDetail09(t *testing.T) {
	in := "hdr v\n\nunterminated-tail"
	var buf bytes.Buffer
	if err := stripHeaderSignatures(&buf, strings.NewReader(in)); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(buf.String(), "unterminated-tail") {
		t.Fatalf("unterminated tail lost: %q", buf.String())
	}
}

// TestDetail10: the destination object's type is the requested objType, not
// the source's.
func TestDetail10(t *testing.T) {
	src := sbObj(t, plumbing.TagObject, "hdr v\n\nbody\n")
	dst := memory.NewStorage().NewEncodedObject()
	if err := stripObjectSignatures(dst, src, plumbing.CommitObject); err != nil {
		t.Fatal(err)
	}
	if dst.Type() != plumbing.CommitObject {
		t.Fatalf("dst type = %v, want CommitObject", dst.Type())
	}
}
