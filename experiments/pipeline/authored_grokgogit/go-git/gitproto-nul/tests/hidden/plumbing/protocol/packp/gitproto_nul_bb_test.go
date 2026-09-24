package packp

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"example.internal/gitkit/v6/plumbing/format/pktline"
)

// gpPayload encodes g and returns the payload of the first pkt-line.
func gpPayload(t *testing.T, g *GitProtoRequest) string {
	t.Helper()
	var buf bytes.Buffer
	if err := g.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	s := pktline.NewScanner(bytes.NewReader(buf.Bytes()))
	if !s.Scan() {
		t.Fatalf("no pkt-line in %q", buf.Bytes())
	}
	return s.Text()
}

func gpDecode(t *testing.T, payload string) (*GitProtoRequest, error) {
	t.Helper()
	var buf bytes.Buffer
	if _, err := pktline.Write(&buf, []byte(payload)); err != nil {
		t.Fatalf("pktline write: %v", err)
	}
	g := &GitProtoRequest{}
	err := g.Decode(&buf)
	return g, err
}

// TestDetail01 (yes): Encode refuses a nil writer with ErrNilWriter.
func TestDetail01(t *testing.T) {
	err := (&GitProtoRequest{RequestCommand: "git-upload-pack", Pathname: "/r"}).Encode(nil)
	if !errors.Is(err, ErrNilWriter) {
		t.Fatalf("Encode(nil) = %v, want ErrNilWriter", err)
	}
}

// TestDetail02 (yes): an empty RequestCommand is invalid.
func TestDetail02(t *testing.T) {
	var buf bytes.Buffer
	err := (&GitProtoRequest{Pathname: "/r"}).Encode(&buf)
	if !errors.Is(err, ErrInvalidGitProtoRequest) {
		t.Fatalf("empty command: err = %v, want ErrInvalidGitProtoRequest", err)
	}
}

// TestDetail03 (shape — Inferable: no): an ASCII control byte in any of
// command, pathname, host, or an extra parameter makes the request invalid.
func TestDetail03(t *testing.T) {
	bad := []byte{0x01, 0x1f, 0x7f}
	good := func() *GitProtoRequest {
		return &GitProtoRequest{
			RequestCommand: "git-upload-pack",
			Pathname:       "/repo.git",
			Host:           "example.com",
			ExtraParams:    []string{"p1"},
		}
	}

	for _, b := range bad {
		for name, mutate := range map[string]func(*GitProtoRequest){
			"command":  func(g *GitProtoRequest) { g.RequestCommand += string(b) },
			"pathname": func(g *GitProtoRequest) { g.Pathname += string(b) },
			"host":     func(g *GitProtoRequest) { g.Host += string(b) },
			"extra":    func(g *GitProtoRequest) { g.ExtraParams[0] += string(b) },
		} {
			g := good()
			mutate(g)
			var buf bytes.Buffer
			err := g.Encode(&buf)
			if !errors.Is(err, ErrInvalidGitProtoRequest) {
				t.Fatalf("byte %#02x in %s: err = %v, want ErrInvalidGitProtoRequest", b, name, err)
			}
		}
	}

	// A clean request is accepted.
	var buf bytes.Buffer
	if err := good().Encode(&buf); err != nil {
		t.Fatalf("clean request rejected: %v", err)
	}
}

// TestDetail04 (shape — Inferable: no): the pkt-line payload is
// `command SP pathname NUL`, then `host=<host> NUL` when host is non-empty,
// then an extra NUL plus `param NUL` per extra parameter when the list is
// non-empty.
func TestDetail04(t *testing.T) {
	cases := []struct {
		g    *GitProtoRequest
		want string
	}{
		{
			&GitProtoRequest{RequestCommand: "git-upload-pack", Pathname: "/r.git"},
			"git-upload-pack /r.git\x00",
		},
		{
			&GitProtoRequest{RequestCommand: "git-upload-pack", Pathname: "/r.git", Host: "h.example"},
			"git-upload-pack /r.git\x00host=h.example\x00",
		},
		{
			&GitProtoRequest{RequestCommand: "git-upload-pack", Pathname: "/r.git", Host: "h.example", ExtraParams: []string{"p1", "p2"}},
			"git-upload-pack /r.git\x00host=h.example\x00\x00p1\x00p2\x00",
		},
		{
			&GitProtoRequest{RequestCommand: "git-upload-pack", Pathname: "/r.git", ExtraParams: []string{"p1"}},
			"git-upload-pack /r.git\x00\x00p1\x00",
		},
	}
	for i, c := range cases {
		if got := gpPayload(t, c.g); got != c.want {
			t.Fatalf("case %d payload = %q, want %q", i, got, c.want)
		}
	}
}

// TestDetail05 (shape — Inferable: no): the extra NUL before extra
// parameters is written even when host was also written, producing
// `...host=h NUL NUL param NUL`.
func TestDetail05(t *testing.T) {
	p := gpPayload(t, &GitProtoRequest{
		RequestCommand: "git-upload-pack",
		Pathname:       "/r.git",
		Host:           "h",
		ExtraParams:    []string{"pa"},
	})
	if !strings.Contains(p, "host=h\x00\x00pa\x00") {
		t.Fatalf("payload = %q, want double NUL between host and params", p)
	}
	if strings.Count(p, "\x00") != 4 {
		t.Fatalf("payload = %q, want 4 NULs (path, host, separator, param)", p)
	}
}

// TestDetail06 (partially): Decode of a flush packet or an empty line is
// io.EOF.
func TestDetail06(t *testing.T) {
	g := &GitProtoRequest{}
	if err := g.Decode(bytes.NewReader([]byte("0000"))); !errors.Is(err, io.EOF) {
		t.Fatalf("flush decode: err = %v, want io.EOF", err)
	}

	// An empty-payload pkt-line is also io.EOF.
	g = &GitProtoRequest{}
	if err := g.Decode(bytes.NewReader([]byte("0004"))); !errors.Is(err, io.EOF) {
		t.Fatalf("empty line decode: err = %v, want io.EOF", err)
	}
}

// TestDetail07 (shape — Inferable: no): the payload must end with a NUL.
func TestDetail07(t *testing.T) {
	g, err := gpDecode(t, "git-upload-pack /r.git")
	if err == nil {
		t.Fatalf("decode of NUL-less payload succeeded: %+v", g)
	}
	if !errors.Is(err, ErrInvalidGitProtoRequest) {
		t.Fatalf("err = %v, want ErrInvalidGitProtoRequest", err)
	}
}

// TestDetail08 (partially): the first space splits command from the rest;
// subsequent fields split on NUL.
func TestDetail08(t *testing.T) {
	g, err := gpDecode(t, "git-upload-pack /r.git\x00host=h\x00")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if g.RequestCommand != "git-upload-pack" || g.Pathname != "/r.git" {
		t.Fatalf("got command=%q path=%q", g.RequestCommand, g.Pathname)
	}

	// A pathname containing a space survives: only the FIRST space splits.
	g, err = gpDecode(t, "git-upload-pack /a b\x00")
	if err != nil {
		t.Fatalf("decode space-path: %v", err)
	}
	if g.RequestCommand != "git-upload-pack" || g.Pathname != "/a b" {
		t.Fatalf("space path: command=%q path=%q", g.RequestCommand, g.Pathname)
	}
}

// TestDetail09 (shape — Inferable: no): Host is the second field with a
// `host=` prefix stripped; the field is not required to carry the prefix.
func TestDetail09(t *testing.T) {
	g, err := gpDecode(t, "git-upload-pack /r\x00host=example.com\x00")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if g.Host != "example.com" {
		t.Fatalf("Host = %q, want example.com", g.Host)
	}

	// No prefix: the raw second field still lands in Host.
	g, err = gpDecode(t, "git-upload-pack /r\x00example.com\x00")
	if err != nil {
		t.Fatalf("decode bare host: %v", err)
	}
	if g.Host != "example.com" {
		t.Fatalf("bare field: Host = %q, want example.com", g.Host)
	}
}

// TestDetail10 (shape — Inferable: no): empty extra-parameter fields are
// dropped.
func TestDetail10(t *testing.T) {
	g, err := gpDecode(t, "git-upload-pack /r\x00host=h\x00\x00p1\x00\x00p2\x00")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.ExtraParams) != 2 || g.ExtraParams[0] != "p1" || g.ExtraParams[1] != "p2" {
		t.Fatalf("ExtraParams = %v, want [p1 p2]", g.ExtraParams)
	}
}

// TestDetail11 (yes): Decode runs the same control-byte check as Encode.
func TestDetail11(t *testing.T) {
	for _, payload := range []string{
		"git-upload-pack /r\x1f.git\x00",
		"git-upload-pack /r.git\x00host=h\x7f\x00",
		"git-upload-pack\x01 /r.git\x00",
	} {
		g, err := gpDecode(t, payload)
		if !errors.Is(err, ErrInvalidGitProtoRequest) {
			t.Fatalf("decode %q: g=%+v err=%v, want ErrInvalidGitProtoRequest", payload, g, err)
		}
	}
}

// TestDetail12 (shape — Inferable: no): invalid requests wrap
// ErrInvalidGitProtoRequest — the sentinel is asserted, never a message.
func TestDetail12(t *testing.T) {
	var buf bytes.Buffer
	for name, err := range map[string]error{
		"empty command":  func() error { return (&GitProtoRequest{Pathname: "/r"}).Encode(&buf) }(),
		"control byte":   func() error { return (&GitProtoRequest{RequestCommand: "c\x01", Pathname: "/r"}).Encode(&buf) }(),
		"decode no NUL":  func() error { _, e := gpDecode(t, "cmd path"); return e }(),
		"decode ctlbyte": func() error { _, e := gpDecode(t, "cmd pa\x02th\x00"); return e }(),
	} {
		if !errors.Is(err, ErrInvalidGitProtoRequest) {
			t.Fatalf("%s: err = %v, want ErrInvalidGitProtoRequest", name, err)
		}
	}
}
