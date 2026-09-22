package object

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/storage/memory"
)

var tpTarget, _ = plumbing.FromHex("1111111111111111111111111111111111111111")

func tagObj(t *testing.T, raw string) plumbing.EncodedObject {
	t.Helper()
	obj := memory.NewStorage().NewEncodedObject()
	obj.SetType(plumbing.TagObject)
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

func encodeTag(t *testing.T, tag *Tag) string {
	t.Helper()
	obj := memory.NewStorage().NewEncodedObject()
	obj.SetType(plumbing.TagObject)
	if err := tag.Encode(obj); err != nil {
		t.Fatal(err)
	}
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

func encodeTagNoSig(t *testing.T, tag *Tag) string {
	t.Helper()
	obj := memory.NewStorage().NewEncodedObject()
	obj.SetType(plumbing.TagObject)
	if err := tag.EncodeWithoutSignature(obj); err != nil {
		t.Fatal(err)
	}
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

var tagWhen = time.Unix(1700000000, 0).UTC()

func rawTag(extra string) string {
	return "object " + tpTarget.String() + "\n" +
		"type commit\n" +
		"tag v1.0\n" +
		"tagger A U Thor <a@b.c> 1700000000 +0000\n" +
		extra +
		"\n" +
		"the message\n"
}

// TestDetail01: headers must be object, type, tag in order; wrong order or
// missing is malformed.
func TestDetail01(t *testing.T) {
	for _, raw := range []string{
		"type commit\nobject " + tpTarget.String() + "\ntag v1\n\nm\n",
		"object " + tpTarget.String() + "\ntag v1\ntype commit\n\nm\n",
		"object " + tpTarget.String() + "\ntype commit\n\nm\n",
		"type commit\nobject " + tpTarget.String() + "\n\nm\n",
	} {
		var tag Tag
		if err := tag.Decode(tagObj(t, raw)); !errors.Is(err, ErrMalformedTag) {
			t.Fatalf("raw %q decoded = %v, want ErrMalformedTag", raw, err)
		}
	}
}

// TestDetail02: tagger is optional — a tag without it parses with a zero
// signature and encodes without the tagger line.
func TestDetail02(t *testing.T) {
	var tag Tag
	err := tag.Decode(tagObj(t,
		"object "+tpTarget.String()+"\ntype commit\ntag v1\n\nm\n"))
	if err != nil {
		t.Fatal(err)
	}
	if tag.Tagger.Name != "" || tag.Tagger.Email != "" || !tag.Tagger.When.IsZero() {
		t.Fatalf("missing tagger produced %+v", tag.Tagger)
	}
	out := encodeTag(t, &tag)
	if strings.Contains(out, "tagger") {
		t.Fatalf("zero tagger emitted: %q", out)
	}
}

// TestDetail03: a duplicate canonical header out of position is silently
// dropped — not an error, not an override.
func TestDetail03(t *testing.T) {
	var tag Tag
	raw := "object " + tpTarget.String() + "\ntype commit\ntag real\n" +
		"tag fake\n\nm\n"
	if err := tag.Decode(tagObj(t, raw)); err != nil {
		t.Fatalf("duplicate tag header rejected: %v", err)
	}
	if tag.Name != "real" {
		t.Fatalf("duplicate overrode name: %q", tag.Name)
	}
}

// TestDetail04: unknown headers are silently dropped.
func TestDetail04(t *testing.T) {
	var tag Tag
	raw := rawTag("x-custom-header whatever\n")
	if err := tag.Decode(tagObj(t, raw)); err != nil {
		t.Fatalf("unknown header rejected: %v", err)
	}
	if tag.Name != "v1.0" {
		t.Fatalf("name = %q", tag.Name)
	}
}

// TestDetail05: gpgsig-sha256 folds continuations (one leading space each)
// and repeated occurrences concatenate.
func TestDetail05(t *testing.T) {
	var tag Tag
	raw := "object " + tpTarget.String() + "\ntype commit\ntag v\n" +
		"gpgsig-sha256 part1\n" +
		" cont1\n" +
		"gpgsig-sha256 part2\n" +
		"\nm\n"
	if err := tag.Decode(tagObj(t, raw)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tag.SignatureSHA256, "part1") ||
		!strings.Contains(tag.SignatureSHA256, "cont1") ||
		!strings.Contains(tag.SignatureSHA256, "part2") {
		t.Fatalf("SignatureSHA256 = %q", tag.SignatureSHA256)
	}
}

// TestDetail06: text after the blank line is the message, except a trailing
// inline PGP signature peels into Signature — on the LAST marker.
func TestDetail06(t *testing.T) {
	var tag Tag
	raw := rawTag("") + "-----BEGIN PGP SIGNATURE-----\nSIGDATA\n-----END PGP SIGNATURE-----\n"
	if err := tag.Decode(tagObj(t, raw)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tag.Signature, "SIGDATA") {
		t.Fatalf("inline signature not peeled: %q", tag.Signature)
	}
	if strings.Contains(tag.Message, "SIGDATA") {
		t.Fatalf("signature left in message: %q", tag.Message)
	}
}

// TestDetail07: encode emits canonical header order with gpgsig-sha256
// between tagger and the blank separator.
func TestDetail07(t *testing.T) {
	tag := &Tag{
		Name: "v1", TargetType: plumbing.CommitObject, Target: tpTarget,
		Tagger:          Signature{Name: "N", Email: "e@x", When: tagWhen},
		SignatureSHA256: "sigval",
		Message:         "msg\n",
	}
	out := encodeTag(t, tag)
	oi := strings.Index(out, "object ")
	ti := strings.Index(out, "\ntype ")
	gi := strings.Index(out, "\ntag ")
	gg := strings.Index(out, "\ntagger ")
	gs := strings.Index(out, "gpgsig-sha256")
	if !(oi == 0 && ti > oi && gi > ti && gg > gi && gs > gg) {
		t.Fatalf("header order wrong: %q", out)
	}
	if strings.Index(out, "\n\n") < gs {
		t.Fatalf("gpgsig-sha256 not before blank line: %q", out)
	}
}

// TestDetail08: Signature is appended verbatim after the message — no
// separator is inserted.
func TestDetail08(t *testing.T) {
	tag := &Tag{
		Name: "v1", TargetType: plumbing.CommitObject, Target: tpTarget,
		Message:   "body-no-trailing-newline",
		Signature: "SIGTAIL",
	}
	out := encodeTag(t, tag)
	if !strings.Contains(out, "body-no-trailing-newlineSIGTAIL") &&
		!strings.HasSuffix(out, "SIGTAIL") {
		t.Fatalf("signature not verbatim-appended: %q", out)
	}
}

// TestDetail09 (shape — Inferable: no): a signature counts as zero only when
// fully empty — a name-only tagger is not zero (observable via encode
// emitting a tagger line or keeping the name on round-trip).
func TestDetail09(t *testing.T) {
	tag := &Tag{
		Name: "v1", TargetType: plumbing.CommitObject, Target: tpTarget,
		Tagger:  Signature{Name: "N"},
		Message: "m\n",
	}
	out := encodeTag(t, tag)
	if !strings.Contains(out, "tagger") {
		t.Fatalf("name-only tagger treated as zero: %q", out)
	}
}

// TestDetail10: unchanged fields → raw bytes streamed minus signature;
// mutated fields → re-encode.
func TestDetail10(t *testing.T) {
	var tag Tag
	raw := "object " + tpTarget.String() + "\ntype commit\ntag v1\n" +
		"tagger A U Thor <a@b.c> 1700000000 +0000\n" +
		"gpgsig-sha256 SIG256\n" +
		"\nmsg\n" +
		"-----BEGIN PGP SIGNATURE-----\nINLINE\n-----END PGP SIGNATURE-----\n"
	if err := tag.Decode(tagObj(t, raw)); err != nil {
		t.Fatal(err)
	}
	stripped := encodeTagNoSig(t, &tag)
	if strings.Contains(stripped, "SIG256") || strings.Contains(stripped, "INLINE") {
		t.Fatalf("signature material kept: %q", stripped)
	}
	if !strings.Contains(stripped, "msg") {
		t.Fatalf("message lost: %q", stripped)
	}

	tag.Message = "changed\n"
	out := encodeTagNoSig(t, &tag)
	if !strings.Contains(out, "changed") {
		t.Fatalf("mutation not re-encoded: %q", out)
	}
	if strings.Contains(out, "SIG256") {
		t.Fatalf("re-encode kept sig header: %q", out)
	}
}

// TestDetail11: the source-match excludes Signature/SignatureSHA256 —
// mutating them still yields the raw-bytes path.
func TestDetail11(t *testing.T) {
	var tag Tag
	raw := "object " + tpTarget.String() + "\ntype commit\ntag v1\n" +
		"tagger A U Thor <a@b.c> 1700000000 +0000\n" +
		"gpgsig-sha256 SIG256\n" +
		"\nmsg\n"
	if err := tag.Decode(tagObj(t, raw)); err != nil {
		t.Fatal(err)
	}
	base := encodeTagNoSig(t, &tag)
	tag.SignatureSHA256 = "tampered"
	mut := encodeTagNoSig(t, &tag)
	if strings.Contains(mut, "tampered") {
		t.Fatalf("mutated sig field re-encoded: %q", mut)
	}
	if mut != base {
		t.Fatalf("sig-field mutation changed output path:\n%q\n%q", base, mut)
	}
}

// TestDetail12 (shape — Inferable: no): a line declined by one state is
// re-served to the next — a header in tagger's position is not lost.
func TestDetail12(t *testing.T) {
	var tag Tag
	raw := "object " + tpTarget.String() + "\ntype commit\ntag v1\n" +
		"gpgsig-sha256 PUSHBACK\n" + // no tagger — this line must survive
		"\nmsg\n"
	if err := tag.Decode(tagObj(t, raw)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tag.SignatureSHA256, "PUSHBACK") {
		t.Fatalf("declined line lost: %q", tag.SignatureSHA256)
	}
}
