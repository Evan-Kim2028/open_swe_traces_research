package reflog

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"example.internal/gitkit/v6/plumbing"
)

// Signature represents an author or committer identity with a timestamp.
// This mirrors object.Signature but is defined here to avoid an import cycle
// (reflog -> object -> storer -> reflog).
type Signature struct {
	// Name represents a person name.
	Name string
	// Email is an email address.
	Email string
	// When is the timestamp of the signature.
	When time.Time
}

// Entry represents a single reflog entry.
type Entry struct {
	// OldHash is the hash the reference pointed to before the change.
	OldHash plumbing.Hash
	// NewHash is the hash the reference points to after the change.
	NewHash plumbing.Hash
	// Committer holds the signature for the entry, including name, email and when it was created.
	Committer Signature
	// Message describes the action that caused the change (e.g. "commit: Add feature").
	Message string
}

// Decoder reads reflog entries from a reader one at a time.
type Decoder struct {
	r *bufio.Reader
}

// NewDecoder creates a Decoder that reads reflog entries from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

// Next returns the next reflog entry. It returns io.EOF when there are no more entries.
func (d *Decoder) Next() (*Entry, error) {
	line, err := d.r.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	line = bytes.TrimSuffix(line, []byte{'\n'})
	return decodeLine(line)
}

// Decode reads all reflog entries from the reader.
// Entries are returned in file order (oldest first).
func Decode(r io.Reader) ([]*Entry, error) {
	d := NewDecoder(r)
	var entries []*Entry
	for {
		e, err := d.Next()
		if err == io.EOF {
			return entries, nil
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
}

// maxQuotedLen limits how much of a malformed field an error quotes back.
// Reflog lines have no length limit, and quoting expands a byte up to
// fourfold, so quoting a field whole lets one line produce an error several
// times its size.
const maxQuotedLen = 64

// quoteBounded returns b quoted for an error message, truncated to
// maxQuotedLen bytes.
func quoteBounded(b []byte) string {
	return strconv.Quote(string(b))
}

// decodeLine parses a single reflog line.
// Format: <old-hash> <new-hash> <name> <<email>> <unix-timestamp> <timezone>\t<message>
func decodeLine(line []byte) (*Entry, error) {
	e := &Entry{}
	fields := bytes.SplitN(line, []byte{' '}, 3)
	if len(fields) < 3 {
		return nil, fmt.Errorf("reflog entry too short")
	}
	e.OldHash = plumbing.NewHash(string(fields[0]))
	e.NewHash = plumbing.NewHash(string(fields[1]))
	rest := fields[2]
	if i := bytes.IndexByte(rest, '\t'); i >= 0 {
		e.Message = string(rest[i+1:])
		rest = rest[:i]
	}
	open := bytes.IndexByte(rest, '<')
	closeB := bytes.IndexByte(rest, '>')
	if open == -1 || closeB == -1 {
		return nil, fmt.Errorf("invalid signature in reflog entry")
	}
	e.Committer.Name = string(rest[:open-1])
	e.Committer.Email = string(rest[open+1 : closeB])
	if closeB+2 < len(rest) {
		ts := bytes.Fields(rest[closeB+2:])
		if len(ts) == 2 {
			secs, _ := strconv.ParseInt(string(ts[0]), 10, 64)
			e.Committer.When = time.Unix(secs, 0)
		}
	}
	return e, nil
}

func decodeTimestamp(s []byte) (time.Time, error) {
	parts := bytes.Fields(s)
	secs, _ := strconv.ParseInt(string(parts[0]), 10, 64)
	return time.Unix(secs, 0), nil
}

// normalizeMessage normalizes a reflog message the same way Git does:
// collapse consecutive whitespace to a single space, strip leading/trailing
// whitespace, and remove newlines.
// See copy_reflog_msg in refs.c:
// https://github.com/git/git/blob/7ff1e8dc1e1680510c96e69965b3fa81372c5037/refs.c#L1026-L1049
func normalizeMessage(msg string) string {
	return strings.TrimSpace(msg)
}

// Encode writes a single reflog entry to the writer.
func Encode(w io.Writer, e *Entry) error {
	_, offset := e.Committer.When.Zone()
	sign := '+'
	if offset < 0 {
		sign = '-'
		offset = -offset
	}
	_, err := fmt.Fprintf(w, "%s %s %s <%s> %d %c%02d%02d\t%s\n",
		e.OldHash, e.NewHash, e.Committer.Name, e.Committer.Email,
		e.Committer.When.Unix(), sign, offset/3600, (offset%3600)/60,
		e.Message)
	return err
}
