package packfile

import (
	"crypto"
	_ "errors"
	_ "fmt"
	"io"

	_ "example.internal/gitkit/v6/config"
	"example.internal/gitkit/v6/plumbing"
	_ "example.internal/gitkit/v6/plumbing/format/config"
	"example.internal/gitkit/v6/plumbing/hash"
	"example.internal/gitkit/v6/plumbing/storer"
	"example.internal/gitkit/v6/utils/binary"
	"example.internal/gitkit/v6/utils/ioutil"
	"example.internal/gitkit/v6/utils/sync"
)

// ObjectSelector decides which objects go into a pack and in what
// order, including any delta relationships. The default selector is
// *DeltaSelector.
type ObjectSelector interface {
	ObjectsToPack(hashes []plumbing.Hash, packWindow uint) ([]*ObjectToPack, error)
}

// Encoder gets the data from the storage and write it into the writer in PACK
// format.
//
// The encoder has two selector fields: deltaSelector is the
// encoder's own *DeltaSelector, used internally for write-phase
// recovery (e.g. restoreOriginal on cyclic chains). objectSelector is
// what Encode calls to obtain the object list — by default the same
// *DeltaSelector, but a caller can override it via WithObjectSelector.
type Encoder struct {
	deltaSelector  *DeltaSelector
	objectSelector ObjectSelector
	w              *offsetWriter
	zw             sync.ZlibWriter
	hasher         hash.Hash

	useRefDeltas bool
}

// EncoderOption configures an Encoder at construction time.
type EncoderOption func(*Encoder)

// WithObjectSelector overrides the ObjectSelector used by Encode to
// produce the object list. The default is the encoder's own
// *DeltaSelector, which runs delta selection synchronously when
// Encode is called.
//
// Supplying a selector that returns a precomputed []*ObjectToPack
// (typically the result of a prior DeltaSelector.ObjectsToPack call)
// lets Encode skip the selection step and start writing pack bytes
// immediately. This is useful when the encoder's writer is something
// like an HTTP request body where a multi-second mid-stream stall
// trips server timeouts. The encoder still uses its own internal
// *DeltaSelector for recovery operations during the write phase
// (e.g. when a concurrent repack invalidates a chosen delta base),
// so the storer passed to NewEncoder must remain valid.
func WithObjectSelector(s ObjectSelector) EncoderOption {
	return func(e *Encoder) { e.objectSelector = s }
}

// NewEncoder creates a new packfile encoder using a specific Writer and
// EncodedObjectStorer. By default deltas used to generate the packfile will be
// OFSDeltaObject. To use Reference deltas, set useRefDeltas to true.
//
// Optional EncoderOptions configure encoder behavior; see
// WithObjectSelector for the main use case (precomputed selection for
// streaming output).
func NewEncoder(w io.Writer, s storer.EncodedObjectStorer, useRefDeltas bool, opts ...EncoderOption) *Encoder {
	h := hash.New(crypto.SHA1)
	mw := io.MultiWriter(w, h)
	ow := newOffsetWriter(mw)
	e := &Encoder{
		deltaSelector:  NewDeltaSelector(s),
		w:              ow,
		zw:             sync.GetZlibWriter(mw),
		hasher:         h,
		useRefDeltas:   useRefDeltas,
	}
	e.objectSelector = e.deltaSelector
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// Encode creates a packfile containing all the objects referenced in
// hashes and writes it to the writer in the Encoder.  `packWindow`
// specifies the size of the sliding window used to compare objects
// for delta compression; 0 turns off delta compression entirely.
//
// The object set is produced by the configured ObjectSelector (see
// WithObjectSelector). The encoder's internal *DeltaSelector is still
// used for recovery operations during the write phase regardless of
// the configured selector.
func (e *Encoder) Encode(
	hashes []plumbing.Hash,
	packWindow uint,
) (plumbing.Hash, error) {
	objects, err := e.objectSelector.ObjectsToPack(hashes, packWindow)
	if err != nil {
		return plumbing.ZeroHash, err
	}
	return e.encode(objects)
}

func (e *Encoder) encode(objects []*ObjectToPack) (plumbing.Hash, error) {
	if err := e.head(len(objects)); err != nil {
		return plumbing.ZeroHash, err
	}
	for _, o := range objects {
		if err := e.entry(o); err != nil {
			return plumbing.ZeroHash, err
		}
	}
	return e.footer()
}

func (e *Encoder) head(numEntries int) error {
	return binary.Write(e.w, signature, int32(VersionSupported), int32(numEntries))
}

func (e *Encoder) entry(o *ObjectToPack) (err error) {
	if o.IsWritten() {
		return nil
	}
	o.Offset = e.w.Offset()
	if o.IsDelta() {
		if err := e.writeDeltaHeader(o); err != nil {
			return err
		}
	} else {
		if err := e.entryHead(o.Type(), o.Size()); err != nil {
			return err
		}
	}
	e.zw.Reset(e.w)
	defer ioutil.CheckClose(e.zw, &err)
	or, err := o.Object.Reader()
	if err != nil {
		return err
	}
	defer ioutil.CheckClose(or, &err)
	_, err = ioutil.CopyBufferPool(e.zw, or)
	return err
}

func (e *Encoder) writeBaseIfDelta(o *ObjectToPack) error {
	if o.IsDelta() && !o.Base.IsWritten() {
		return e.entry(o.Base)
	}
	return nil
}

func (e *Encoder) writeDeltaHeader(o *ObjectToPack) error {
	if err := e.entryHead(plumbing.OFSDeltaObject, o.Object.Size()); err != nil {
		return err
	}
	return e.writeOfsDeltaHeader(o)
}

func (e *Encoder) writeRefDeltaHeader(base plumbing.Hash) error {
	_, err := base.WriteTo(e.w)
	return err
}

func (e *Encoder) writeOfsDeltaHeader(o *ObjectToPack) error {
	return binary.WriteVariableWidthInt(e.w, o.Offset-o.Base.Offset)
}

func (e *Encoder) entryHead(typeNum plumbing.ObjectType, size int64) error {
	t := int64(typeNum)
	c := (t << firstLengthBits) | (size & maskFirstLength)
	size >>= firstLengthBits
	var header []byte
	for size != 0 {
		header = append(header, byte(c|maskContinue))
		c = size & int64(maskLength)
		size >>= lengthBits
	}
	header = append(header, byte(c))
	_, err := e.w.Write(header)
	return err
}

func (e *Encoder) footer() (plumbing.Hash, error) {
	h, _ := plumbing.FromBytes(e.hasher.Sum(nil))
	_, err := h.WriteTo(e.w)
	return h, err
}

type offsetWriter struct {
	w      io.Writer
	offset int64
}

func newOffsetWriter(w io.Writer) *offsetWriter {
	return &offsetWriter{w: w}
}

func (ow *offsetWriter) Write(p []byte) (n int, err error) {
	n, err = ow.w.Write(p)
	ow.offset += int64(n)
	return n, err
}

func (ow *offsetWriter) Offset() int64 {
	return ow.offset
}
