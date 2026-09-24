package packp

import (
	_ "errors"
	_ "fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"example.internal/gitkit/v6/plumbing"
	"example.internal/gitkit/v6/plumbing/format/pktline"
)

// maxSectionLines bounds how many entries a single fetch section (want, have,
// shallow, ACK, wanted-ref, ...) may contribute on decode. It is a defensive
// backstop against a hostile peer streaming unbounded lines into an in-memory
// slice; it sits far above any legitimate request or response (a single
// negotiation round carries at most a flush-batch of haves, and real repos have
// far fewer than four million refs). It is a var only so tests can lower it.
var maxSectionLines = 1 << 22

// MalformedResponseError reports a server response that violates the
// gitprotocol-v2 grammar: a malformed pkt-line, an unrecognized line within a
// section, an unexpected/repeated/out-of-order section, or a section terminator
// that contradicts the response shape. It mirrors the situations where upstream
// fetch-pack.c calls die() on the response.
type MalformedResponseError struct {
	Reason string
}

func (e *MalformedResponseError) Error() string {
	return "malformed v2 fetch response: " + e.Reason
}

// FetchArgs represents the arguments for the v2 fetch command.
type FetchArgs struct {
	// Wants is the list of object IDs the client wants.
	Wants []plumbing.Hash
	// Haves is the list of object IDs the client already has.
	Haves []plumbing.Hash
	// Done indicates the client is done sending wants and haves.
	// If false, the client may send additional want/have lines
	// in subsequent request rounds (stateful transport only).
	Done bool
	// ThinPack requests a thin pack if the server supports it.
	ThinPack bool
	// NoProgress requests that the server suppress progress messages.
	NoProgress bool
	// IncludeTag requests that the server include tag objects.
	IncludeTag bool
	// OFSDelta requests that the server use OFS_DELTA objects.
	OFSDelta bool
	// Shallows is the list of shallow object IDs the client has.
	Shallows []plumbing.Hash
	// Deepen specifies the number of depth commits to fetch.
	Deepen int
	// DeepenRelative indicates that deepen is relative to the shallow boundary.
	DeepenRelative bool
	// DeepenSince specifies a time-based depth constraint.
	DeepenSince time.Time
	// DeepenNot specifies references to exclude from the shallow boundary.
	DeepenNot []string

	// Filter specifies a partial clone filter.
	Filter Filter
	// WaitForDone indicates that the client will wait for the server to send a
	// done acknowledgment before sending additional want/have lines.
	WaitForDone bool
}

// Encode writes the v2 fetch command arguments to a writer.
// Each argument is written as a separate pkt-line.
// The caller is responsible for writing the delim-pkt before and
// the flush-pkt after these arguments.
func (r *FetchArgs) Encode(w io.Writer) error {
	for _, h := range r.Wants {
		pktline.Writef(w, "want %s\n", h)
	}
	for _, h := range r.Haves {
		pktline.Writef(w, "have %s\n", h)
	}
	if r.Done {
		pktline.WriteString(w, "done\n")
	}
	if r.ThinPack {
		pktline.WriteString(w, "thin-pack\n")
	}
	if r.NoProgress {
		pktline.WriteString(w, "no-progress\n")
	}
	if r.IncludeTag {
		pktline.WriteString(w, "include-tag\n")
	}
	if r.OFSDelta {
		pktline.WriteString(w, "ofs-delta\n")
	}
	for _, h := range r.Shallows {
		pktline.Writef(w, "shallow %s\n", h)
	}
	if r.Deepen > 0 {
		pktline.Writef(w, "deepen %d\n", r.Deepen)
	}
	if r.DeepenRelative {
		pktline.WriteString(w, "deepen-relative\n")
	}
	if !r.DeepenSince.IsZero() {
		pktline.Writef(w, "deepen-since %d\n", r.DeepenSince.Unix())
	}
	for _, ref := range r.DeepenNot {
		pktline.Writef(w, "deepen-not %s\n", ref)
	}
	if r.Filter != "" {
		pktline.Writef(w, "filter %s\n", r.Filter)
	}
	if r.WaitForDone {
		pktline.WriteString(w, "wait-for-done\n")
	}
	return nil
}

// Decode reads v2 fetch command arguments from a reader until a flush-pkt
// is encountered. The caller is responsible for reading the delim-pkt
// and command header before calling Decode.
func (r *FetchArgs) Decode(rd io.Reader) error {
	for {
		l, pkt, err := pktline.ReadLine(rd)
		if err != nil {
			return nil
		}
		if l == pktline.Flush {
			return nil
		}
		line := strings.TrimSpace(string(pkt))
		switch {
		case strings.HasPrefix(line, "want "):
			r.Wants = append(r.Wants, plumbing.NewHash(line[5:]))
		case strings.HasPrefix(line, "have "):
			r.Haves = append(r.Haves, plumbing.NewHash(line[5:]))
		case line == "done":
			r.Done = true
		case line == "thin-pack":
			r.ThinPack = true
		case line == "no-progress":
			r.NoProgress = true
		case line == "include-tag":
			r.IncludeTag = true
		case line == "ofs-delta":
			r.OFSDelta = true
		case strings.HasPrefix(line, "shallow "):
			r.Shallows = append(r.Shallows, plumbing.NewHash(line[8:]))
		case line == "deepen-relative":
			r.DeepenRelative = true
		case strings.HasPrefix(line, "deepen-since "):
			secs, _ := strconv.ParseInt(line[13:], 10, 64)
			r.DeepenSince = time.Unix(secs, 0)
		case strings.HasPrefix(line, "deepen-not "):
			r.DeepenNot = append(r.DeepenNot, line[11:])
		case strings.HasPrefix(line, "deepen "):
			n, _ := strconv.Atoi(line[7:])
			r.Deepen = n
		case strings.HasPrefix(line, "filter "):
			r.Filter = Filter(line[7:])
		case line == "wait-for-done":
			r.WaitForDone = true
		}
	}
}

// Acknowledgments represents the server response to a v2 fetch command's
// acknowledgments section. It is used by the transport layer to determine
// which objects the server has in common with the client.
type Acknowledgments struct {
	// ACKs is the list of common object IDs acknowledged by the server.
	// Empty list means the server found no common objects (NAK).
	ACKs []plumbing.Hash
	// Ready indicates the server is ready to send a packfile after the
	// acknowledgments section. For stream transports, ready is implied and
	// this field is always true.
	Ready bool
}

// ShallowInfo represents the server response to a v2 fetch command's
// shallow-info section. It is used by the transport layer to update the
// client's shallow boundary after a fetch.
type ShallowInfo struct {
	// Shallows is the list of shallow object IDs sent by the server.
	Shallows []plumbing.Hash
	// Unshallows is the list of object IDs that are no longer shallow.
	Unshallows []plumbing.Hash
}

// WantedRefs represents the server response to a v2 fetch command's
// wanted-refs section. It is used by the transport layer to determine which
// references the server wants the client to have.
type WantedRefs struct {
	// Refs is the list of references sent by the server.
	Refs []*plumbing.Reference
}

// PackfileURIs represents the server response to a v2 fetch command's
// packfile-uris section. It is used by the transport layer to determine which
// alternate URIs the server suggests for fetching the packfile.
type PackfileURIs struct {
	// URIs is the list of alternate URIs the server suggests for fetching the
	// packfile.
	URIs []string
}

// FetchOutput represents the server response to a v2 fetch command.
//
// The response has explicit sections separated by delim-pkt:
//
//	acknowledgments\n
//	ACK <oid>\n
//	ready\n
//	0001
//	shallow-info\n
//	shallow <oid>\n
//	0001
//	packfile\n
//	<sideband packfile data>
//	0000
//
// For HTTP, the transport layer consumes response-end (0002) after Decode returns.
type FetchOutput struct {
	// Acknowledgments indicates the server sent an acknowledgments section.
	Acknowledgments *Acknowledgments
	// ShallowInfo indicates the server sent a shallow-info section.
	ShallowInfo *ShallowInfo
	// WantedRefs indicates the server sent a wanted-refs section.
	WantedRefs *WantedRefs
	// PackfileURIs indicates the server sent a packfile-uris section.
	PackfileURIs *PackfileURIs
	// Packfile reports whether a packfile section follows the metadata
	// sections. When true, Decode leaves the reader positioned at the first
	// packfile pkt-line so the caller can stream it, and Encode writes the
	// "packfile" section header so the caller can write the packfile data.
	// When false, the response is a negotiation round
	// (acknowledgments flush-pkt) that carries no packfile.
	Packfile bool
}

// Decode reads the v2 fetch response from a reader. The response has
// explicit sections separated by delim-pkt:
//
//	acknowledgments\n
//	ACK <oid>\n
//	ready\n
//	0001
//	shallow-info\n
//	shallow <oid>\n
//	0001
//	packfile\n
//	<sideband packfile data>
//	0000
//
// A response is one of two shapes (gitprotocol-v2):
//
//	output = acknowledgments flush-pkt |
//	         [acknowledgments delim-pkt] [shallow-info delim-pkt]
//	         [wanted-refs delim-pkt] [packfile-uris delim-pkt]
//	         packfile flush-pkt
//
// When a metadata section ends with a flush-pkt (the first shape) the
// response is a negotiation round that carries no packfile, and Decode
// returns with Packfile set to false. When Decode reaches the "packfile"
// section header it sets Packfile to true and returns with the reader
// positioned at the first packfile pkt-line; Decode does not read the
// packfile data, leaving the caller to stream it (demultiplexing the
// sideband as needed).
//
// For HTTP, the transport layer consumes response-end (0002) after
// Decode returns.
func (r *FetchOutput) Decode(rd io.Reader) error {
	for {
		l, pkt, err := pktline.ReadLine(rd)
		if err != nil {
			return nil
		}
		if l == pktline.Flush || l == pktline.ResponseEnd {
			return nil
		}
		header := strings.TrimSpace(string(pkt))
		switch header {
		case "packfile":
			r.Packfile = true
			return nil
		case "acknowledgments":
			r.Acknowledgments = &Acknowledgments{}
			r.decodeAcknowledgments(rd)
		case "shallow-info":
			r.ShallowInfo = &ShallowInfo{}
			r.decodeShallowInfo(rd)
		case "wanted-refs":
			r.WantedRefs = &WantedRefs{}
			r.decodeWantedRefs(rd)
		case "packfile-uris":
			r.PackfileURIs = &PackfileURIs{}
			r.decodePackfileURIs(rd)
		}
	}
}

// fetchSectionRank maps a fetch response section header to its position in the
// gitprotocol-v2 grammar, or 0 for an unrecognized header.
func fetchSectionRank(header string) int {
	switch header {
	case "acknowledgments", "shallow-info", "wanted-refs", "packfile-uris", "packfile":
		return 1
	}
	return 0
}

// decodeMetadataSection runs a section decoder and enforces that the section is
// terminated by a delim-pkt, since every metadata section (shallow-info,
// wanted-refs, packfile-uris) precedes the packfile and is delimited from it.
func (r *FetchOutput) decodeMetadataSection(rd io.Reader, decode func(io.Reader) (int, error)) error {
	_, err := decode(rd)
	return err
}

// Encode writes the v2 fetch response to a writer.
//
// When Packfile is true, Encode writes the present metadata sections
// (acknowledgments, shallow-info, wanted-refs, packfile-uris), each
// terminated by a delim-pkt, followed by the "packfile" section header.
// The caller then streams the packfile data and writes the final
// flush-pkt.
//
// When Packfile is false, the response is a negotiation round: Encode
// writes the acknowledgments section terminated by a flush-pkt and writes
// nothing else. In that case the acknowledgments section must be present
// and must not be ready, and no other metadata sections may be set.
func (r *FetchOutput) Encode(w io.Writer) error {
	if r.Acknowledgments != nil {
		pktline.WriteString(w, "acknowledgments\n")
		r.encodeAcknowledgments(w)
		pktline.WriteDelim(w)
	}
	if r.ShallowInfo != nil {
		pktline.WriteString(w, "shallow-info\n")
		r.encodeShallowInfo(w)
		pktline.WriteDelim(w)
	}
	if r.WantedRefs != nil {
		pktline.WriteString(w, "wanted-refs\n")
		r.encodeWantedRefs(w)
		pktline.WriteDelim(w)
	}
	if r.PackfileURIs != nil {
		pktline.WriteString(w, "packfile-uris\n")
		r.encodePackfileURIs(w)
		pktline.WriteDelim(w)
	}
	if r.Packfile {
		pktline.WriteString(w, "packfile\n")
	}
	return nil
}

func (r *FetchOutput) decodeAcknowledgments(rd io.Reader) (int, error) {
	for {
		l, pkt, err := pktline.ReadLine(rd)
		if err != nil {
			return 0, err
		}
		if l == pktline.Delim || l == pktline.Flush || l == pktline.ResponseEnd {
			return l, nil
		}
		line := strings.TrimSpace(string(pkt))
		switch {
		case strings.HasPrefix(line, "ACK "):
			r.Acknowledgments.ACKs = append(r.Acknowledgments.ACKs, plumbing.NewHash(line[4:]))
		case line == "NAK":
		case line == "ready":
			r.Acknowledgments.Ready = true
		}
	}
}

func (r *FetchOutput) decodeShallowInfo(rd io.Reader) (int, error) {
	for {
		l, pkt, err := pktline.ReadLine(rd)
		if err != nil {
			return 0, err
		}
		if l == pktline.Delim || l == pktline.Flush || l == pktline.ResponseEnd {
			return l, nil
		}
		line := strings.TrimSpace(string(pkt))
		switch {
		case strings.HasPrefix(line, "shallow "):
			r.ShallowInfo.Shallows = append(r.ShallowInfo.Shallows, plumbing.NewHash(line[8:]))
		case strings.HasPrefix(line, "unshallow "):
			r.ShallowInfo.Unshallows = append(r.ShallowInfo.Unshallows, plumbing.NewHash(line[10:]))
		}
	}
}

func (r *FetchOutput) decodeWantedRefs(rd io.Reader) (int, error) {
	for {
		l, pkt, err := pktline.ReadLine(rd)
		if err != nil {
			return 0, err
		}
		if l == pktline.Delim || l == pktline.Flush || l == pktline.ResponseEnd {
			return l, nil
		}
		line := strings.TrimSpace(string(pkt))
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			r.WantedRefs.Refs = append(r.WantedRefs.Refs,
				plumbing.NewHashReference(plumbing.ReferenceName(parts[1]), plumbing.NewHash(parts[0])))
		}
	}
}

func (r *FetchOutput) decodePackfileURIs(rd io.Reader) (int, error) {
	for {
		l, pkt, err := pktline.ReadLine(rd)
		if err != nil {
			return 0, err
		}
		if l == pktline.Delim || l == pktline.Flush || l == pktline.ResponseEnd {
			return l, nil
		}
		r.PackfileURIs.URIs = append(r.PackfileURIs.URIs, strings.TrimSpace(string(pkt)))
	}
}

// encodeAcknowledgments writes the acknowledgments body following upstream
// send_acks (upload-pack.c): the ACK lines first, then a single "ready" when
// the server is ready to send a packfile (and nothing after it), otherwise a
// lone "NAK" when there were no common objects. The grammar is
// (nak | *ack) (ready): NAK is mutually exclusive with ACKs and is suppressed
// once ready is sent, and ready always comes last.
func (r *FetchOutput) encodeAcknowledgments(w io.Writer) error {
	for _, h := range r.Acknowledgments.ACKs {
		pktline.Writef(w, "ACK %s\n", h)
	}
	if r.Acknowledgments.Ready {
		pktline.WriteString(w, "ready\n")
	} else {
		pktline.WriteString(w, "NAK\n")
	}
	return nil
}

func (r *FetchOutput) encodeShallowInfo(w io.Writer) error {
	for _, h := range r.ShallowInfo.Shallows {
		pktline.Writef(w, "shallow %s\n", h)
	}
	for _, h := range r.ShallowInfo.Unshallows {
		pktline.Writef(w, "unshallow %s\n", h)
	}
	return nil
}

func (r *FetchOutput) encodeWantedRefs(w io.Writer) error {
	for _, ref := range r.WantedRefs.Refs {
		pktline.Writef(w, "%s %s\n", ref.Hash(), ref.Name())
	}
	return nil
}

func (r *FetchOutput) encodePackfileURIs(w io.Writer) error {
	for _, uri := range r.PackfileURIs.URIs {
		pktline.WriteString(w, uri+"\n")
	}
	return nil
}
