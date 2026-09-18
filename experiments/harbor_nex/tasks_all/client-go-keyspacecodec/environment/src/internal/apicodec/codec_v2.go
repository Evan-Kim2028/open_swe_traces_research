package apicodec

import (
	"encoding/binary"

	"github.com/pingcap/kvproto/pkg/errorpb"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pkg/errors"
	"github.com/tikv/client-go/v2/tikvrpc"
)

var (
	// DefaultKeyspaceID is the keyspaceID of the default keyspace.
	DefaultKeyspaceID uint32 = 0
	// DefaultKeyspaceName is the name of the default keyspace.
	DefaultKeyspaceName = "DEFAULT"

	rawModePrefix     byte = 'r'
	txnModePrefix     byte = 'x'
	keyspacePrefixLen      = 4

	// maxKeyspaceID is the maximum value of keyspaceID, its value is uint24Max.
	maxKeyspaceID = uint32(0xFFFFFF)

	// errKeyOutOfBound happens when key to be decoded lies outside the keyspace's range.
	errKeyOutOfBound = errors.New("given key does not belong to the keyspace")
)

func checkV2Key(b []byte) error {
	if len(b) < keyspacePrefixLen || (b[0] != rawModePrefix && b[0] != txnModePrefix) {
		return errors.Errorf("invalid API V2 key %s", b)
	}
	return nil
}

// BuildKeyspaceName builds a keyspace name
func BuildKeyspaceName(name string) string {
	if name == "" {
		return DefaultKeyspaceName
	}
	return name
}

// codecV2 is used to encode/decode keys and request into APIv2 format.
type codecV2 struct {
	keyspaceID KeyspaceID
	prefix     []byte
	endKey     []byte
	memCodec   memCodec
}

// NewCodecV2 returns a codec that can be used to encode/decode
// keys and requests to and from APIv2 format.
func NewCodecV2(mode Mode, keyspaceID uint32) (Codec, error) {
	if keyspaceID > maxKeyspaceID {
		return nil, errors.Errorf("keyspaceID %d is out of range, maximum is %d", keyspaceID, maxKeyspaceID)
	}
	prefix, err := getIDByte(keyspaceID)
	if err != nil {
		return nil, err
	}
	codec := &codecV2{
		keyspaceID: KeyspaceID(keyspaceID),
		// Region keys in CodecV2 are always encoded in memory comparable form.
		memCodec: &memComparableCodec{},
	}
	codec.prefix = make([]byte, 4)
	codec.endKey = make([]byte, 4)
	switch mode {
	case ModeRaw:
		codec.prefix[0] = rawModePrefix
	case ModeTxn:
		codec.prefix[0] = txnModePrefix
	default:
		return nil, errors.Errorf("unknown mode")
	}
	copy(codec.prefix[1:], prefix)
	prefixVal := binary.BigEndian.Uint32(codec.prefix)
	binary.BigEndian.PutUint32(codec.endKey, prefixVal+1)
	return codec, nil
}

func getIDByte(keyspaceID uint32) ([]byte, error) {
	// PutUint32 requires 4 bytes to operate, so must use buffer with size 4 here.
	b := make([]byte, 4)
	// Use BigEndian to put the least significant byte to last array position.
	// For example, keyspaceID 1 should result in []byte{0, 0, 1}
	binary.BigEndian.PutUint32(b, keyspaceID)
	// When keyspaceID can't fit in 3 bytes, first byte of buffer will be non-zero.
	// So return error.
	if b[0] != 0 {
		return nil, errors.Errorf("illegal keyspaceID: %v, keyspaceID must be 3 byte", b)
	}
	// Remove the first byte to make keyspace ID 3 bytes.
	return b[1:], nil
}

func (c *codecV2) GetKeyspace() []byte {
	return c.prefix
}

func (c *codecV2) GetKeyspaceID() KeyspaceID {
	return c.keyspaceID
}

func (c *codecV2) GetAPIVersion() kvrpcpb.APIVersion {
	return kvrpcpb.APIVersion_V2
}

// EncodeRequest encodes with the given Codec.
func (c *codecV2) EncodeRequest(req *tikvrpc.Request) (*tikvrpc.Request, error) {
	return req, nil
}

func (c *codecV2) DecodeResponse(req *tikvrpc.Request, resp *tikvrpc.Response) (*tikvrpc.Response, error) {
	return resp, nil
}

func (c *codecV2) EncodeRegionKey(key []byte) []byte {
	return key
}

func (c *codecV2) DecodeRegionKey(encodedKey []byte) ([]byte, error) {
	return encodedKey, nil
}

func (c *codecV2) EncodeRegionRange(start, end []byte) ([]byte, []byte) {
	return start, end
}

func (c *codecV2) DecodeRegionRange(encodedStart, encodedEnd []byte) ([]byte, []byte, error) {
	return encodedStart, encodedEnd, nil
}

func (c *codecV2) EncodeRange(start, end []byte) ([]byte, []byte) {
	return start, end
}

func (c *codecV2) encodeRange(start, end []byte, reverse bool) ([]byte, []byte) {
	return start, end
}

func (c *codecV2) DecodeRange(encodedStart, encodedEnd []byte) (start []byte, end []byte, err error) {
	return encodedStart, encodedEnd, nil
}

func (c *codecV2) EncodeKey(key []byte) []byte {
	return key
}

func (c *codecV2) DecodeKey(encodedKey []byte) ([]byte, error) {
	return encodedKey, nil
}

func (c *codecV2) encodeKeyRanges(keyRanges []*kvrpcpb.KeyRange) []*kvrpcpb.KeyRange {
	return keyRanges
}

func (c *codecV2) decodeRegionError(regionError *errorpb.Error) (*errorpb.Error, error) {
	return regionError, nil
}

func (c *codecV2) DecodeBucketKeys(keys [][]byte) ([][]byte, error) {
	return keys, nil
}
