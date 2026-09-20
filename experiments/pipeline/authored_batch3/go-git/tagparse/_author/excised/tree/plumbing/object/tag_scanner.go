package object

import (
	"bufio"
	"bytes"
	_ "fmt"
	_ "io"

	_ "example.internal/gitkit/v6/plumbing"
)

// tagScanner holds the working state of the tag decoder driven by the
// stateFn loop in (*Tag).Decode. Each tagState reads one or more lines
// from r, updates the in-progress *Tag and the scanner's bookkeeping,
// and returns the state that should run next (or nil to stop).
type tagScanner struct {
	r      *bufio.Reader
	t      *Tag
	msgbuf bytes.Buffer

	// pending holds a line that was read but the current state decided to
	// hand back to the next state, paired with the io.EOF flag returned
	// when the line was originally read.
	pending    []byte
	pendingErr error

	// First-occurrence tracking — once the corresponding canonical
	// header has been decoded at its expected position, subsequent
	// occurrences (or out-of-position lines) are silently dropped,
	// matching the strict layout enforced by upstream's
	// parse_tag_buffer (tag.c:130).
	//
	// gpgsig-sha256 is NOT tracked here: upstream's
	// parse_buffer_signed_by_header (commit.c:1186) accumulates every
	// occurrence into one signature buffer, so we do the same.
	sawObject, sawType, sawName, sawTagger bool
}

// tagState is one step of the decoder state machine. Each function reads
// the lines it needs, mutates *Tag via s.t, and returns the next state
// to run (or nil to terminate the loop).
type tagState func(*tagScanner) (tagState, error)

// readLine returns the next line from the buffer, transparently
// consuming any line that was previously pushed back by a state that
// decided not to handle it.
func (s *tagScanner) readLine() ([]byte, error) {
	panic("excised: tagScanner.readLine")
}

// pushBack stashes an unconsumed line so the next state's readLine call
// sees it. Only one line can be pushed back at a time.
func (s *tagScanner) pushBack(line []byte, err error) {
	panic("excised: tagScanner.pushBack")
}

// scanTagObject requires the first line to be `object HASH`, mirroring
// upstream's strict parse_tag_buffer (tag.c:151-156). Anything else
// returns ErrMalformedTag.
func scanTagObject(s *tagScanner) (tagState, error) {
	panic("excised: scanTagObject")
}

// scanTagType requires a `type` line immediately after the object header,
// mirroring upstream's parse_tag_buffer (tag.c:158-166).
func scanTagType(s *tagScanner) (tagState, error) {
	panic("excised: scanTagType")
}

// scanTagName requires a `tag` line immediately after the type header,
// mirroring upstream's parse_tag_buffer (tag.c:186-194).
func scanTagName(s *tagScanner) (tagState, error) {
	panic("excised: scanTagName")
}

// scanTagTagger accepts a `tagger` line at its canonical position. Any
// other header is pushed back for scanTagHeaders.
func scanTagTagger(s *tagScanner) (tagState, error) {
	panic("excised: scanTagTagger")
}

// scanTagHeaders dispatches one header line. gpgsig-sha256 hands off to
// scanTagPgp256Cont so the continuation block can be consumed; out-of-
// canonical-position fields and unknown headers are silently dropped.
func scanTagHeaders(s *tagScanner) (tagState, error) {
	panic("excised: scanTagHeaders")
}

// scanTagPgp256Cont accumulates continuation lines for the gpgsig-sha256
// header. Continuations strip exactly one leading space, mirroring
// upstream's `line + 1` (commit.c:1509). The first non-continuation line
// is pushed back so scanTagHeaders can dispatch it — repeat occurrences
// of the same header land back here and concatenate, matching upstream's
// parse_buffer_signed_by_header (commit.c:1186).
func scanTagPgp256Cont(s *tagScanner) (tagState, error) {
	panic("excised: scanTagPgp256Cont")
}

// scanTagMessage drains the remaining bytes into the message buffer.
// (*Tag).Decode then runs parseSignedBytes over those bytes to peel off
// the optional inline trailing PGP signature.
func scanTagMessage(s *tagScanner) (tagState, error) {
	panic("excised: scanTagMessage")
}
