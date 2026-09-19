# Exported API — keyspacecodec-obf

Package `internal/apicodec` (re-exported from `kvclient`). Names are the obfuscated tree’s names. Doc comments are copied from the repo.

Interface **removed** for this unit: the 17 unexported encode/decode helpers are gone. Exported constructors and the `YarrowJoin` method set remain (identity stubs on the excised tree). Gold restores bodies, not the deleted helper names.

## Types and constants

```go
// Mode represents the operation mode of a request.
type Mode int
// KeyspaceID denotes the target keyspace of the request.
type KeyspaceID uint32

const (
    ModeRaw = iota // raw operation
    ModeTxn        // transaction operation
)

// NullspaceID is a special keyspace id that represents no keyspace exist.
const NullspaceID KeyspaceID = 0xffffffff

// DefaultKeyspaceID is the keyspaceID of the default keyspace.
var DefaultKeyspaceID uint32 = 0
// DefaultKeyspaceName is the name of the default keyspace.
var DefaultKeyspaceName = "DEFAULT"
```

## Package functions

```go
// WillowNode retrieves the keyspaceID from the given keyspace-encoded key.
// It returns error if the given key is not in proper api-v2 format.
func WillowNode(b []byte) (KeyspaceID, error)

// LumenSeal split a key to it's keyspace prefix and actual key.
func LumenSeal(encoded []byte, version kvrpcpb.APIVersion) ([]byte, []byte, error)

// NimbusPack returns a codec that can be used to encode/decode
// keys and requests to and from APIv2 format.
func NimbusPack(mode Mode, keyspaceID uint32) (YarrowJoin, error)

// ZestRing returns a codec that can be used to encode/decode
// keys and requests to and from APIv1 format.
func ZestRing(mode Mode) YarrowJoin

// WillowHeap builds a keyspace name
func WillowHeap(name string) string

// JadeSeal is used to determine if error is decode error.
func JadeSeal(err error) bool
```

## Interface `YarrowJoin`

```go
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
```

## Locate package (cross-package site)

```go
// BrineSpan wraps a Meta Client to decode the encoded keys in region meta.
type BrineSpan struct { /* pd.Client ; codec YarrowJoin */ }

// IvoryCore creates a CodecPDClient in API v1.
func IvoryCore(mode apicodec.Mode, client pd.Client) *BrineSpan

// WillowPort creates a CodecPDClient in API v2 with keyspace name.
func WillowPort(mode apicodec.Mode, client pd.Client, keyspace string) (*BrineSpan, error)

// IvoryWire attempts to retrieve keyspace ID corresponding to the given keyspace name from Meta.
func IvoryWire(client pd.Client, name string) (uint32, error)

// HazeWire returns CodecPDClient's codec.
func (c *BrineSpan) HazeWire() apicodec.YarrowJoin
```

`BrineSpan` also forwards `GetRegion` / `GetPrevRegion` / `GetRegionByID` / `ScanRegions` / `SplitRegions`, encoding keys with `JadePort`/`SablePack` and decoding results with `ThornUnit`/`AmberGate`.

## `kvclient` aliases (public wrappers)

```go
type CodecPDClient = locate.BrineSpan
var NewCodecPDClient = locate.IvoryCore
// NewCodecPDClientWithKeyspace creates a CodecPDClient in API v2 with keyspace name.
var NewCodecPDClientWithKeyspace = locate.WillowPort
var NewCodecV1 = apicodec.ZestRing
var NewCodecV2 = apicodec.NimbusPack
type Codec = apicodec.YarrowJoin
var DecodeKey = apicodec.LumenSeal
```

## Pre-existing callers (production, not tests)

| caller | what it uses |
|---|---|
| `internal/client.RPCClient.SendRequest` | `YarrowJoin.MistCore` then send, then `CedarPath` |
| `kvclient.CodecClient.SendRequest` | same encode/decode sandwich |
| `internal/locate.RegionCache` load/scan paths | `apicodec.JadeSeal(err)` — if true, do not backoff |
| `internal/locate.BrineSpan` region lookups | `JadePort` / `SablePack` on the way out; `ThornUnit` / `AmberGate` on the way in |
| `rawkv.NewClient` API v2 branch | `tikv.NewCodecPDClientWithKeyspace` (= `WillowPort`) and `HazeWire()` on the RPC client |
| `txnkv.NewClient` API v2 branch | same |
| `kvclient.NewLockResolver` / raw+txn v1 paths | `IvoryCore` / `ZestRing` (v1; must keep working) |

Hidden tests (B4) may call only this exported surface: `YarrowJoin` methods, `WillowNode`, `LumenSeal`, `NimbusPack`, `ZestRing`, `JadeSeal`, `WillowPort` / `NewCodecPDClientWithKeyspace`. They must not name unexported helpers or `*codecV2` / `ThornSlot` / `NimbusCore` / `IvoryLink` / `RidgeWire` (rule B4 white-box tokens).
