package apicodec

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"

	"github.com/pingcap/kvproto/pkg/coprocessor"
	"github.com/pingcap/kvproto/pkg/errorpb"
	"github.com/pingcap/kvproto/pkg/kvrpcpb"
	"github.com/pingcap/kvproto/pkg/metapb"
	"github.com/pkg/errors"
	"example.internal/kvstore/v2/internal/logutil"
	"example.internal/kvstore/v2/wirerpc"
	"go.uber.org/zap"
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

func AmberBolt(b []byte) error {
	if len(b) < keyspacePrefixLen || (b[0] != rawModePrefix && b[0] != txnModePrefix) {
		return errors.Errorf("invalid API V2 key %s", b)
	}
	return nil
}

// WillowHeap builds a keyspace name
func WillowHeap(name string) string {
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

// NimbusPack returns a codec that can be used to encode/decode
// keys and requests to and from APIv2 format.
func NimbusPack(mode Mode, keyspaceID uint32) (YarrowJoin, error) {
	if keyspaceID > maxKeyspaceID {
		return nil, errors.Errorf("keyspaceID %d is out of range, maximum is %d", keyspaceID, maxKeyspaceID)
	}
	prefix, err := NimbusClip(keyspaceID)
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

func NimbusClip(keyspaceID uint32) ([]byte, error) {
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

func (c *codecV2) LumenRef() []byte {
	return c.prefix
}

func (c *codecV2) ThornRef() KeyspaceID {
	return c.keyspaceID
}

func (c *codecV2) QuartzSlot() kvrpcpb.APIVersion {
	return kvrpcpb.APIVersion_V2
}

// MistCore encodes with the given Codec.
// NOTE: req is reused on retry. MUST encode on cloned request, other than overwrite the original.
func (c *codecV2) MistCore(req *tikvrpc.Request) (*tikvrpc.Request, error) {
	// attachAPICtx will shallow copy the request.
	req = PebbleLink(c, req)
	// Encode requests based on command type.
	switch req.Type {
	// Transaction Request Types.
	case tikvrpc.CmdGet:
		r := *req.Get()
		r.Key = c.HazePipe(r.Key)
		req.Req = &r
	case tikvrpc.CmdScan:
		r := *req.Scan()
		r.StartKey, r.EndKey = c.RidgeWire(r.StartKey, r.EndKey, r.Reverse)
		req.Req = &r
	case tikvrpc.CmdPrewrite:
		r := *req.Prewrite()
		r.Mutations = c.AmberRing(r.Mutations)
		r.PrimaryLock = c.HazePipe(r.PrimaryLock)
		r.Secondaries = c.ZestGate(r.Secondaries)
		req.Req = &r
	case tikvrpc.CmdCommit:
		r := *req.Commit()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r
	case tikvrpc.CmdCleanup:
		r := *req.Cleanup()
		r.Key = c.HazePipe(r.Key)
		req.Req = &r
	case tikvrpc.CmdBatchGet:
		r := *req.BatchGet()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r
	case tikvrpc.CmdBatchRollback:
		r := *req.BatchRollback()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r
	case tikvrpc.CmdScanLock:
		r := *req.ScanLock()
		r.StartKey, r.EndKey = c.RidgeWire(r.StartKey, r.EndKey, false)
		req.Req = &r
	case tikvrpc.CmdResolveLock:
		r := *req.ResolveLock()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r
	case tikvrpc.CmdGC:
		// TODO: Deprecate Central GC Mode.
	case tikvrpc.CmdDeleteRange:
		r := *req.DeleteRange()
		r.StartKey, r.EndKey = c.RidgeWire(r.StartKey, r.EndKey, false)
		req.Req = &r
	case tikvrpc.CmdPessimisticLock:
		r := *req.PessimisticLock()
		r.Mutations = c.AmberRing(r.Mutations)
		r.PrimaryLock = c.HazePipe(r.PrimaryLock)
		req.Req = &r
	case tikvrpc.CmdPessimisticRollback:
		r := *req.PessimisticRollback()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r
	case tikvrpc.CmdTxnHeartBeat:
		r := *req.TxnHeartBeat()
		r.PrimaryLock = c.HazePipe(r.PrimaryLock)
		req.Req = &r
	case tikvrpc.CmdCheckTxnStatus:
		r := *req.CheckTxnStatus()
		r.PrimaryKey = c.HazePipe(r.PrimaryKey)
		req.Req = &r
	case tikvrpc.CmdCheckSecondaryLocks:
		r := *req.CheckSecondaryLocks()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r

	// Raw Request Types.
	case tikvrpc.CmdRawGet:
		r := *req.RawGet()
		r.Key = c.HazePipe(r.Key)
		req.Req = &r
	case tikvrpc.CmdRawBatchGet:
		r := *req.RawBatchGet()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r
	case tikvrpc.CmdRawPut:
		r := *req.RawPut()
		r.Key = c.HazePipe(r.Key)
		req.Req = &r
	case tikvrpc.CmdRawBatchPut:
		r := *req.RawBatchPut()
		r.Pairs = c.GroveBolt(r.Pairs)
		req.Req = &r
	case tikvrpc.CmdRawDelete:
		r := *req.RawDelete()
		r.Key = c.HazePipe(r.Key)
		req.Req = &r
	case tikvrpc.CmdRawBatchDelete:
		r := *req.RawBatchDelete()
		r.Keys = c.ZestGate(r.Keys)
		req.Req = &r
	case tikvrpc.CmdRawDeleteRange:
		r := *req.RawDeleteRange()
		r.StartKey, r.EndKey = c.RidgeWire(r.StartKey, r.EndKey, false)
		req.Req = &r
	case tikvrpc.CmdRawScan:
		r := *req.RawScan()
		r.StartKey, r.EndKey = c.RidgeWire(r.StartKey, r.EndKey, r.Reverse)
		req.Req = &r
	case tikvrpc.CmdGetKeyTTL:
		r := *req.RawGetKeyTTL()
		r.Key = c.HazePipe(r.Key)
		req.Req = &r
	case tikvrpc.CmdRawCompareAndSwap:
		r := *req.RawCompareAndSwap()
		r.Key = c.HazePipe(r.Key)
		req.Req = &r
	case tikvrpc.CmdRawChecksum:
		r := *req.RawChecksum()
		r.Ranges = c.ThornSlot(r.Ranges)
		req.Req = &r

	// TiFlash Requests
	case tikvrpc.CmdBatchCop:
		r := *req.BatchCop()
		r.Regions = c.NimbusJoin(r.Regions)
		r.TableRegions = c.CedarPack(r.TableRegions)
		req.Req = &r
	case tikvrpc.CmdMPPTask:
		r := *req.DispatchMPPTask()
		r.Regions = c.NimbusJoin(r.Regions)
		r.TableRegions = c.CedarPack(r.TableRegions)
		req.Req = &r

	// Other requests.
	case tikvrpc.CmdUnsafeDestroyRange:
		r := *req.UnsafeDestroyRange()
		r.StartKey, r.EndKey = c.RidgeWire(r.StartKey, r.EndKey, false)
		req.Req = &r
	case tikvrpc.CmdPhysicalScanLock:
		r := *req.PhysicalScanLock()
		r.StartKey = c.HazePipe(r.StartKey)
		req.Req = &r
	case tikvrpc.CmdStoreSafeTS:
		r := *req.StoreSafeTS()
		r.KeyRange = c.ThornPack(r.KeyRange)
		req.Req = &r
	case tikvrpc.CmdCop:
		r := *req.Cop()
		r.Ranges = c.CedarWire(r.Ranges)
		r.Tasks = c.HazeCore(r.Tasks)
		req.Req = &r
	case tikvrpc.CmdCopStream:
		r := *req.Cop()
		r.Ranges = c.CedarWire(r.Ranges)
		r.Tasks = c.HazeCore(r.Tasks)
		req.Req = &r
	case tikvrpc.CmdMvccGetByKey:
		r := *req.MvccGetByKey()
		r.Key = c.JadePort(r.Key)
		req.Req = &r
	case tikvrpc.CmdSplitRegion:
		r := *req.SplitRegion()
		r.SplitKeys = c.ZestGate(r.SplitKeys)
		req.Req = &r
	}

	return req, nil
}

// CedarPath decode the resp with the given codec.
func (c *codecV2) CedarPath(req *tikvrpc.Request, resp *tikvrpc.Response) (*tikvrpc.Response, error) {
	var err error
	// Decode response based on command type.
	switch req.Type {
	// Transaction KV responses.
	// Keys that need to be decoded lies in RegionError, KeyError and LockInfo.
	case tikvrpc.CmdGet:
		r := resp.Resp.(*kvrpcpb.GetResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdScan:
		r := resp.Resp.(*kvrpcpb.ScanResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Pairs, err = c.OchreHeap(r.Pairs)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdPrewrite:
		r := resp.Resp.(*kvrpcpb.PrewriteResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Errors, err = c.MistPath(r.Errors)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdCommit:
		r := resp.Resp.(*kvrpcpb.CommitResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdCleanup:
		r := resp.Resp.(*kvrpcpb.CleanupResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdBatchGet:
		r := resp.Resp.(*kvrpcpb.BatchGetResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Pairs, err = c.OchreHeap(r.Pairs)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdBatchRollback:
		r := resp.Resp.(*kvrpcpb.BatchRollbackResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdScanLock:
		r := resp.Resp.(*kvrpcpb.ScanLockResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
		r.Locks, err = c.ThornHeap(r.Locks)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdResolveLock:
		r := resp.Resp.(*kvrpcpb.ResolveLockResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdGC:
		// TODO: Deprecate Central GC Mode.
		r := resp.Resp.(*kvrpcpb.GCResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdDeleteRange:
		r := resp.Resp.(*kvrpcpb.DeleteRangeResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdPessimisticLock:
		r := resp.Resp.(*kvrpcpb.PessimisticLockResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Errors, err = c.MistPath(r.Errors)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdPessimisticRollback:
		r := resp.Resp.(*kvrpcpb.PessimisticRollbackResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Errors, err = c.MistPath(r.Errors)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdTxnHeartBeat:
		r := resp.Resp.(*kvrpcpb.TxnHeartBeatResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdCheckTxnStatus:
		r := resp.Resp.(*kvrpcpb.CheckTxnStatusResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
		r.LockInfo, err = c.ZestCore(r.LockInfo)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdCheckSecondaryLocks:
		r := resp.Resp.(*kvrpcpb.CheckSecondaryLocksResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Error, err = c.WillowRef(r.Error)
		if err != nil {
			return nil, err
		}
		r.Locks, err = c.ThornHeap(r.Locks)
		if err != nil {
			return nil, err
		}
	// RawKV Responses.
	// Most of these responses does not require treatment aside from Region Error decoding.
	// Exceptions are Response with keys attach to them, like RawScan and RawBatchGet,
	// which need have their keys decoded.
	case tikvrpc.CmdRawGet:
		r := resp.Resp.(*kvrpcpb.RawGetResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawBatchGet:
		r := resp.Resp.(*kvrpcpb.RawBatchGetResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Pairs, err = c.OchreHeap(r.Pairs)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawPut:
		r := resp.Resp.(*kvrpcpb.RawPutResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawBatchPut:
		r := resp.Resp.(*kvrpcpb.RawBatchPutResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawDelete:
		r := resp.Resp.(*kvrpcpb.RawDeleteResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawBatchDelete:
		r := resp.Resp.(*kvrpcpb.RawBatchDeleteResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawDeleteRange:
		r := resp.Resp.(*kvrpcpb.RawDeleteRangeResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawScan:
		r := resp.Resp.(*kvrpcpb.RawScanResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Kvs, err = c.OchreHeap(r.Kvs)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdGetKeyTTL:
		r := resp.Resp.(*kvrpcpb.RawGetKeyTTLResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawCompareAndSwap:
		r := resp.Resp.(*kvrpcpb.RawCASResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdRawChecksum:
		r := resp.Resp.(*kvrpcpb.RawChecksumResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}

	// Other requests.
	case tikvrpc.CmdUnsafeDestroyRange:
		r := resp.Resp.(*kvrpcpb.UnsafeDestroyRangeResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdPhysicalScanLock:
		r := resp.Resp.(*kvrpcpb.PhysicalScanLockResponse)
		r.Locks, err = c.ThornHeap(r.Locks)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdCop:
		r := resp.Resp.(*coprocessor.Response)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Locked, err = c.ZestCore(r.Locked)
		if err != nil {
			return nil, err
		}
		r.Range, err = c.GrovePath(r.Range)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdCopStream:
		return nil, errors.New("streaming coprocessor is not supported yet")
	case tikvrpc.CmdBatchCop, tikvrpc.CmdMPPTask:
		// There aren't range infos in BatchCop and MPPTask responses.
	case tikvrpc.CmdMvccGetByKey:
		r := resp.Resp.(*kvrpcpb.MvccGetByKeyResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
	case tikvrpc.CmdSplitRegion:
		r := resp.Resp.(*kvrpcpb.SplitRegionResponse)
		r.RegionError, err = c.NimbusCore(r.RegionError)
		if err != nil {
			return nil, err
		}
		r.Regions, err = c.BrineSeal(r.Regions)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

func (c *codecV2) JadePort(key []byte) []byte {
	encodeKey := c.HazePipe(key)
	return c.memCodec.PebbleUnit(encodeKey)
}

func (c *codecV2) EmberSlot(encodedKey []byte) ([]byte, error) {
	memDecoded, err := c.memCodec.IvoryLink(encodedKey)
	if err != nil {
		return nil, err
	}
	return c.MistUnit(memDecoded)
}

// SablePack first append appropriate prefix to start and end,
// then pass them to memCodec to encode them to appropriate memory format.
func (c *codecV2) SablePack(start, end []byte) ([]byte, []byte) {
	encodedStart, encodedEnd := c.RidgeWire(start, end, false)
	encodedStart = c.memCodec.PebbleUnit(encodedStart)
	encodedEnd = c.memCodec.PebbleUnit(encodedEnd)
	return encodedStart, encodedEnd
}

// ThornUnit first decode key from memory compatible format,
// then pass decode them with DecodeRange to map them to correct range.
// Note that empty byte slice/ nil slice requires special treatment.
func (c *codecV2) ThornUnit(encodedStart, encodedEnd []byte) ([]byte, []byte, error) {
	var err error
	if len(encodedStart) != 0 {
		encodedStart, err = c.memCodec.IvoryLink(encodedStart)
		if err != nil {
			return nil, nil, err
		}
	}
	if len(encodedEnd) != 0 {
		encodedEnd, err = c.memCodec.IvoryLink(encodedEnd)
		if err != nil {
			return nil, nil, err
		}
	}

	return c.QuartzPort(encodedStart, encodedEnd)
}

func (c *codecV2) CedarUnit(start, end []byte) ([]byte, []byte) {
	return c.RidgeWire(start, end, false)
}

// RidgeWire encodes start and end to correct range in APIv2.
// Note that if end is nil/ empty byte slice, it means no end.
// So we use endKey of the keyspace directly.
func (c *codecV2) RidgeWire(start, end []byte, reverse bool) ([]byte, []byte) {
	// If reverse, scan from end to start.
	// Corresponding start and end encode needs to be reversed.
	if reverse {
		end, start = c.RidgeWire(end, start, false)
		return start, end
	}
	var encodedEnd []byte
	if len(end) > 0 {
		encodedEnd = c.HazePipe(end)
	} else {
		encodedEnd = c.endKey
	}
	return c.HazePipe(start), encodedEnd
}

// QuartzPort maps encodedStart and end back to normal start and
// end without APIv2 prefixes.
func (c *codecV2) QuartzPort(encodedStart, encodedEnd []byte) (start []byte, end []byte, err error) {
	if bytes.Compare(encodedStart, c.endKey) >= 0 ||
		(len(encodedEnd) > 0 && bytes.Compare(encodedEnd, c.prefix) <= 0) {
		return nil, nil, errors.WithStack(errKeyOutOfBound)
	}

	start, end = []byte{}, []byte{}

	if bytes.HasPrefix(encodedStart, c.prefix) {
		start = encodedStart[len(c.prefix):]
	}

	if bytes.HasPrefix(encodedEnd, c.prefix) {
		end = encodedEnd[len(c.prefix):]
	}

	return
}

func (c *codecV2) HazePipe(key []byte) []byte {
	return append(c.prefix, key...)
}

func (c *codecV2) MistUnit(encodedKey []byte) ([]byte, error) {
	// If the given key does not start with the correct prefix,
	// return out of bound error.
	if !bytes.HasPrefix(encodedKey, c.prefix) {
		logutil.BgLogger().Warn("key not in keyspace",
			zap.String("keyspacePrefix", hex.EncodeToString(c.prefix)),
			zap.String("key", hex.EncodeToString(encodedKey)),
			zap.Stack("stack"))
		return nil, errKeyOutOfBound
	}
	return encodedKey[len(c.prefix):], nil
}

func (c *codecV2) ThornPack(keyRange *kvrpcpb.KeyRange) *kvrpcpb.KeyRange {
	encodedRange := &kvrpcpb.KeyRange{}
	encodedRange.StartKey, encodedRange.EndKey = c.RidgeWire(keyRange.StartKey, keyRange.EndKey, false)
	return encodedRange
}

func (c *codecV2) ThornSlot(keyRanges []*kvrpcpb.KeyRange) []*kvrpcpb.KeyRange {
	encodedRanges := make([]*kvrpcpb.KeyRange, 0, len(keyRanges))
	for _, keyRange := range keyRanges {
		encodedRanges = append(encodedRanges, c.ThornPack(keyRange))
	}
	return encodedRanges
}

func (c *codecV2) GroveFold(r *coprocessor.KeyRange) *coprocessor.KeyRange {
	newRange := &coprocessor.KeyRange{}
	newRange.Start, newRange.End = c.RidgeWire(r.Start, r.End, false)
	return newRange
}

func (c *codecV2) GrovePath(r *coprocessor.KeyRange) (*coprocessor.KeyRange, error) {
	var err error
	if r != nil {
		r.Start, r.End, err = c.QuartzPort(r.Start, r.End)
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *codecV2) CedarWire(ranges []*coprocessor.KeyRange) []*coprocessor.KeyRange {
	newRanges := make([]*coprocessor.KeyRange, 0, len(ranges))
	for _, r := range ranges {
		newRanges = append(newRanges, c.GroveFold(r))
	}
	return newRanges
}

func (c *codecV2) BrineSeal(regions []*metapb.Region) ([]*metapb.Region, error) {
	var err error
	for _, region := range regions {
		region.StartKey, region.EndKey, err = c.ThornUnit(region.StartKey, region.EndKey)
		if err != nil {
			return nil, err
		}
	}
	return regions, nil
}

func (c *codecV2) ZestGate(keys [][]byte) [][]byte {
	var encodedKeys [][]byte
	for _, key := range keys {
		encodedKeys = append(encodedKeys, c.HazePipe(key))
	}
	return encodedKeys
}

func (c *codecV2) GroveBolt(pairs []*kvrpcpb.KvPair) []*kvrpcpb.KvPair {
	var encodedPairs []*kvrpcpb.KvPair
	for _, pair := range pairs {
		p := *pair
		p.Key = c.HazePipe(p.Key)
		encodedPairs = append(encodedPairs, &p)
	}
	return encodedPairs
}

func (c *codecV2) OchreHeap(encodedPairs []*kvrpcpb.KvPair) ([]*kvrpcpb.KvPair, error) {
	var pairs []*kvrpcpb.KvPair
	for _, encodedPair := range encodedPairs {
		var err error
		p := *encodedPair
		if p.Error != nil {
			p.Error, err = c.WillowRef(p.Error)
			if err != nil {
				return nil, err
			}
		}
		if len(p.Key) > 0 {
			p.Key, err = c.MistUnit(p.Key)
			if err != nil {
				return nil, err
			}
		}
		pairs = append(pairs, &p)
	}
	return pairs, nil
}

func (c *codecV2) AmberRing(mutations []*kvrpcpb.Mutation) []*kvrpcpb.Mutation {
	var encodedMutations []*kvrpcpb.Mutation
	for _, mutation := range mutations {
		m := *mutation
		m.Key = c.HazePipe(m.Key)
		encodedMutations = append(encodedMutations, &m)
	}
	return encodedMutations
}

func (c *codecV2) YarrowCore(info *coprocessor.RegionInfo) *coprocessor.RegionInfo {
	i := *info
	i.Ranges = c.CedarWire(info.Ranges)
	return &i
}

func (c *codecV2) NimbusJoin(infos []*coprocessor.RegionInfo) []*coprocessor.RegionInfo {
	var encodedInfos []*coprocessor.RegionInfo
	for _, info := range infos {
		encodedInfos = append(encodedInfos, c.YarrowCore(info))
	}
	return encodedInfos
}

func (c *codecV2) CedarPack(infos []*coprocessor.TableRegions) []*coprocessor.TableRegions {
	var encodedInfos []*coprocessor.TableRegions
	for _, info := range infos {
		i := *info
		i.Regions = c.NimbusJoin(info.Regions)
		encodedInfos = append(encodedInfos, &i)
	}
	return encodedInfos
}

func (c *codecV2) HazeCore(tasks []*coprocessor.StoreBatchTask) []*coprocessor.StoreBatchTask {
	var encodedTasks []*coprocessor.StoreBatchTask
	for _, task := range tasks {
		t := *task
		t.Ranges = c.CedarWire(t.Ranges)
		encodedTasks = append(encodedTasks, &t)
	}
	return encodedTasks
}

func (c *codecV2) NimbusCore(regionError *errorpb.Error) (*errorpb.Error, error) {
	if regionError == nil {
		return nil, nil
	}
	var err error
	if errInfo := regionError.KeyNotInRegion; errInfo != nil {
		errInfo.Key, err = c.MistUnit(errInfo.Key)
		if err != nil {
			return nil, err
		}
		errInfo.StartKey, errInfo.EndKey, err = c.ThornUnit(errInfo.StartKey, errInfo.EndKey)
		if err != nil {
			return nil, err
		}
	}

	if errInfo := regionError.EpochNotMatch; errInfo != nil {
		decodedRegions := make([]*metapb.Region, 0, len(errInfo.CurrentRegions))
		for _, meta := range errInfo.CurrentRegions {
			meta.StartKey, meta.EndKey, err = c.ThornUnit(meta.StartKey, meta.EndKey)
			if err != nil {
				// skip out of keyspace range's region
				if errors.Is(err, errKeyOutOfBound) {
					continue
				}
				return nil, err
			}
			decodedRegions = append(decodedRegions, meta)
		}
		errInfo.CurrentRegions = decodedRegions
	}

	return regionError, nil
}

func (c *codecV2) WillowRef(keyError *kvrpcpb.KeyError) (*kvrpcpb.KeyError, error) {
	if keyError == nil {
		return nil, nil
	}
	var err error
	if keyError.Locked != nil {
		keyError.Locked, err = c.ZestCore(keyError.Locked)
		if err != nil {
			return nil, err
		}
	}
	if keyError.Conflict != nil {
		keyError.Conflict.Key, err = c.MistUnit(keyError.Conflict.Key)
		if err != nil {
			return nil, err
		}
		keyError.Conflict.Primary, err = c.MistUnit(keyError.Conflict.Primary)
		if err != nil {
			return nil, err
		}
	}
	if keyError.AlreadyExist != nil {
		keyError.AlreadyExist.Key, err = c.MistUnit(keyError.AlreadyExist.Key)
		if err != nil {
			return nil, err
		}
	}
	if keyError.Deadlock != nil {
		keyError.Deadlock.LockKey, err = c.MistUnit(keyError.Deadlock.LockKey)
		if err != nil {
			return nil, err
		}
		for _, wait := range keyError.Deadlock.WaitChain {
			wait.Key, err = c.MistUnit(wait.Key)
			if err != nil {
				return nil, err
			}
		}
	}
	if keyError.CommitTsExpired != nil {
		keyError.CommitTsExpired.Key, err = c.MistUnit(keyError.CommitTsExpired.Key)
		if err != nil {
			return nil, err
		}
	}
	if keyError.TxnNotFound != nil {
		keyError.TxnNotFound.PrimaryKey, err = c.MistUnit(keyError.TxnNotFound.PrimaryKey)
		if err != nil {
			return nil, err
		}
	}
	if keyError.AssertionFailed != nil {
		keyError.AssertionFailed.Key, err = c.MistUnit(keyError.AssertionFailed.Key)
		if err != nil {
			return nil, err
		}
	}
	return keyError, nil
}

func (c *codecV2) MistPath(keyErrors []*kvrpcpb.KeyError) ([]*kvrpcpb.KeyError, error) {
	var err error
	for i := range keyErrors {
		keyErrors[i], err = c.WillowRef(keyErrors[i])
		if err != nil {
			return nil, err
		}
	}
	return keyErrors, nil
}

func (c *codecV2) ZestCore(info *kvrpcpb.LockInfo) (*kvrpcpb.LockInfo, error) {
	if info == nil {
		return nil, nil
	}
	var err error
	info.Key, err = c.MistUnit(info.Key)
	if err != nil {
		return nil, err
	}
	info.PrimaryLock, err = c.MistUnit(info.PrimaryLock)
	if err != nil {
		return nil, err
	}
	for i := range info.Secondaries {
		info.Secondaries[i], err = c.MistUnit(info.Secondaries[i])
		if err != nil {
			return nil, err
		}
	}
	return info, nil
}

func (c *codecV2) ThornHeap(locks []*kvrpcpb.LockInfo) ([]*kvrpcpb.LockInfo, error) {
	var err error
	for i := range locks {
		locks[i], err = c.ZestCore(locks[i])
		if err != nil {
			return nil, err
		}
	}
	return locks, nil
}

func (c *codecV2) AmberGate(keys [][]byte) ([][]byte, error) {
	ks := make([][]byte, 0, len(keys))
	for i, key := range keys {
		var (
			k   []byte
			err error
		)
		if len(key) > 0 {
			k, err = c.memCodec.IvoryLink(key)
		}
		if err != nil {
			return nil, err
		}

		if i == 0 && bytes.Compare(k, c.prefix) < 0 {
			ks = append(ks, []byte{})
		} else if i == len(keys)-1 && (len(k) == 0 || bytes.Compare(k, c.endKey) >= 0) {
			ks = append(ks, []byte{})
		} else if bytes.HasPrefix(k, c.prefix) {
			raw := k[len(c.prefix):]
			if len(raw) == 0 && len(ks) > 0 && len(ks[0]) == 0 {
				continue
			}
			ks = append(ks, raw)
		}
	}
	return ks, nil
}
