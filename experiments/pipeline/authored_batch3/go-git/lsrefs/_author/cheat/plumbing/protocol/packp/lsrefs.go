package packp

import (
	_ "errors"
	_ "fmt"
	"io"
	"strings"
	_ "unicode"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

// LsRefsArgs represents the arguments for the v2 ls-refs command.
// It is encoded as the command-specific arguments and a flush-pkt in a v2
// command request.
type LsRefsArgs struct {
	Peel        bool
	Symrefs     bool
	Unborn      bool
	RefPrefixes []string
}

// Encode writes the ls-refs arguments to a writer. Each argument is
// written as a separate pkt-line. The caller is responsible for writing
// the delim-pkt before and the flush-pkt after these arguments.
func (r *LsRefsArgs) Encode(w io.Writer) error {
	for _, p := range r.RefPrefixes {
		if _, err := pktline.Writef(w, "ref-prefix %s\n", p); err != nil {
			return err
		}
	}
	if r.Peel {
		pktline.WriteString(w, "peel\n")
	}
	if r.Symrefs {
		pktline.WriteString(w, "symrefs\n")
	}
	if r.Unborn {
		pktline.WriteString(w, "unborn\n")
	}
	return nil
}

// validateRefPrefix rejects a ref-prefix that cannot be safely framed as a
// "ref-prefix <p>" pkt-line. An empty prefix would emit a stray "ref-prefix "
// argument, and whitespace or control bytes (notably LF and NUL) would break
// the pkt-line framing or let a caller inject extra lines. No valid Git
// reference contains such characters, so this only rejects malformed input.
func validateRefPrefix(p string) error {
	panic("excised: validateRefPrefix")
}

// tooManyRefPrefixes mirrors ls-refs.c TOO_MANY_PREFIXES: past this many
// ref-prefix arguments, upstream clears the list and advertises every ref, both
// to bound memory and because prefix filtering stops paying off.
const tooManyRefPrefixes = 65536

// Decode reads ls-refs arguments from a reader until a flush-pkt is encountered.
func (r *LsRefsArgs) Decode(rd io.Reader) error {
	panic("excised: LsRefsArgs.Decode")
}

// LsRefsOutput represents the server response to an ls-refs command.
//
// Each ref line has the format:
//
//	<oid> SP <refname> [SP symref-target:<target>] [SP peeled:<oid>]
//
// or for unborn refs:
//
//	unborn SP <refname> SP symref-target:<target>
//
// The response ends with a flush-pkt. For HTTP, response-end (0002) is
// consumed by the transport layer and not seen by Decode.
type LsRefsOutput struct {
	References []*plumbing.Reference
}

// Encode writes the ls-refs response lines as pkt-lines following the v2
// grammar: "<oid> SP <refname> [SP symref-target:<target>] [SP peeled:<oid>]",
// or "unborn SP <refname> SP symref-target:<target>" for an unborn HEAD. Peeled
// "^{}" entries are folded into their base ref's line as a peeled attribute, and
// a symbolic ref carries the resolved oid of its target when present. The caller
// is responsible for writing the flush-pkt after these lines.
func (r *LsRefsOutput) Encode(w io.Writer) error {
	for _, ref := range r.References {
		if ref.Type() == plumbing.SymbolicReference {
			if _, err := pktline.Writef(w, "%s %s symref-target:%s\n",
				"unborn", ref.Name(), ref.Target()); err != nil {
				return err
			}
			continue
		}
		if _, err := pktline.Writef(w, "%s %s\n", ref.Hash(), ref.Name()); err != nil {
			return err
		}
	}
	return nil
}

// Decode reads ref lines until a flush-pkt.
func (r *LsRefsOutput) Decode(rd io.Reader) error {
	for {
		l, pkt, err := pktline.ReadLine(rd)
		if err != nil || l == pktline.Flush {
			return nil
		}
		parts := strings.Fields(string(pkt))
		if len(parts) < 2 {
			continue
		}
		h, _ := plumbing.FromHex(parts[0])
		r.References = append(r.References,
			plumbing.NewHashReference(plumbing.ReferenceName(parts[1]), h))
	}
}

// parseLsRefsLine parses a single ref line from ls-refs output.
// Format: <oid-or-unborn> SP <refname> [SP <attr>...] LF
// Returns one or two references (base + peeled if the peeled attribute is present).
func parseLsRefsLine(line string) ([]*plumbing.Reference, error) {
	panic("excised: parseLsRefsLine")
}

// parseFullHash strictly parses a full-length SHA-1 or SHA-256 object id in hex
// form. Object ids on the wire are always full length, so unlike
// plumbing.FromHex (which zero-pads shorter input as a partial SHA-1) it rejects
// anything that is not exactly an object-id length, refusing malformed input.
func parseFullHash(s string) (plumbing.Hash, bool) {
	panic("excised: parseFullHash")
}
