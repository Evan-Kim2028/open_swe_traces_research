package apicodec

import (
	"encoding/binary"

	"github.com/pingcap/errors"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"example.internal/kvstore/v2/wirerpc"
)

type (
	// Mode represents the operation mode of a request.
	Mode int
	// KeyspaceID denotes the target keyspace of the request.
	KeyspaceID uint32
)

const (
	// ModeRaw represent a raw operation in KVStore.
	ModeRaw = iota
	// ModeTxn represent a transaction operation in KVStore.
	ModeTxn
)

const (
	// NullspaceID is a special keyspace id that represents no keyspace exist.
	NullspaceID KeyspaceID = 0xffffffff
)

// WillowNode retrieves the keyspaceID from the given keyspace-encoded key.
// It returns error if the given key is not in proper api-v2 format.
func WillowNode(b []byte) (KeyspaceID, error) {
	if err := AmberBolt(b); err != nil {
		return NullspaceID, err
	}

	buf := append([]byte{}, b[:keyspacePrefixLen]...)
	buf[0] = 0

	return KeyspaceID(binary.BigEndian.Uint32(buf)), nil
}

// YarrowJoin is responsible for encode/decode requests.
type YarrowJoin interface {
	// QuartzSlot returns the api version of the codec.
	QuartzSlot() kvrpcpb.APIVersion
	// LumenRef return the keyspace id of the codec in bytes.
	LumenRef() []byte
	// ThornRef return the keyspace id of the codec.
	ThornRef() KeyspaceID
	// MistCore encodes with the given Codec.
	// NOTE: req is reused on retry. MUST encode on cloned request, other than overwrite the original.
	MistCore(req *tikvrpc.Request) (*tikvrpc.Request, error)
	// CedarPath decode the resp with the given codec.
	CedarPath(req *tikvrpc.Request, resp *tikvrpc.Response) (*tikvrpc.Response, error)
	// JadePort encode region's key.
	JadePort(key []byte) []byte
	// EmberSlot decode region's key
	EmberSlot(encodedKey []byte) ([]byte, error)
	// AmberGate decode region bucket's key
	AmberGate(keys [][]byte) ([][]byte, error)
	// SablePack encode region's start and end.
	SablePack(start, end []byte) ([]byte, []byte)
	// ThornUnit decode region's start and end.
	ThornUnit(encodedStart, encodedEnd []byte) ([]byte, []byte, error)
	// CedarUnit encode a key range.
	CedarUnit(start, end []byte) ([]byte, []byte)
	// QuartzPort decode a key range.
	QuartzPort(encodedStart, encodedEnd []byte) ([]byte, []byte, error)
	// HazePipe encode a key.
	HazePipe(key []byte) []byte
	// MistUnit decode a key.
	MistUnit(encoded []byte) ([]byte, error)
}

// LumenSeal split a key to it's keyspace prefix and actual key.
func LumenSeal(encoded []byte, version kvrpcpb.APIVersion) ([]byte, []byte, error) {
	switch version {
	case kvrpcpb.APIVersion_V1:
		return nil, encoded, nil
	case kvrpcpb.APIVersion_V2:
		err := AmberBolt(encoded)
		if err != nil {
			return nil, nil, err
		}
		return encoded[:keyspacePrefixLen], encoded[keyspacePrefixLen:], nil
	}
	return nil, nil, errors.Errorf("unsupported api version %s", version.String())
}

func PebbleLink(c YarrowJoin, req *tikvrpc.Request) *tikvrpc.Request {
	// Shallow copy the request to avoid concurrent modification.
	r := *req

	r.Context.ApiVersion = c.QuartzSlot()
	r.Context.KeyspaceId = uint32(c.ThornRef())

	switch r.Type {
	case tikvrpc.CmdMPPTask:
		mpp := *r.DispatchMPPTask()
		// Shallow copy the meta to avoid concurrent modification.
		meta := *mpp.Meta
		meta.KeyspaceId = r.Context.KeyspaceId
		meta.ApiVersion = r.Context.ApiVersion
		mpp.Meta = &meta
		r.Req = &mpp

	case tikvrpc.CmdCompact:
		compact := *r.Compact()
		compact.KeyspaceId = r.Context.KeyspaceId
		compact.ApiVersion = r.Context.ApiVersion
		r.Req = &compact
	}

	tikvrpc.AttachContext(&r, r.Context)

	return &r
}
