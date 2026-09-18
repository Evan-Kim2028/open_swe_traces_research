"""One-shot: write iter9 bug/alt/cheat patches against client-go, then restore."""

from __future__ import annotations

import subprocess
from pathlib import Path

ROOT = Path("/home/evan/Documents/open_swe_traces_research")
REPO = ROOT / "experiments/codegraph_bugs/repos/client-go"
BUGS = ROOT / "experiments/codegraph_bugs/bugs/client-go"
BUGS.mkdir(parents=True, exist_ok=True)


def git(*args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", "-C", str(REPO), *args],
        capture_output=True,
        text=True,
        check=check,
    )


def restore(*rels: str) -> None:
    git("checkout", "--", *rels)


CODEC_V2_STUB_TAIL = r'''
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
'''


def write_codec_v2_stub() -> None:
    path = REPO / "internal/apicodec/codec_v2.go"
    text = path.read_text()
    marker = "func (c *codecV2) GetKeyspace() []byte {"
    idx = text.find(marker)
    if idx < 0:
        raise SystemExit("codec_v2.go: GetKeyspace marker missing")
    # Keep package, imports, consts, NewCodecV2, getIDByte. Drop unused imports later.
    head = text[:idx]
    # Shrink imports: drop bytes/hex/coprocessor/metapb/logutil/zap if unused in stub.
    head = head.replace('\t"bytes"\n', "")
    head = head.replace('\t"encoding/hex"\n', "")
    head = head.replace('\t"github.com/pingcap/kvproto/pkg/coprocessor"\n', "")
    head = head.replace('\t"github.com/pingcap/kvproto/pkg/metapb"\n', "")
    head = head.replace('\t"github.com/tikv/client-go/v2/internal/logutil"\n', "")
    head = head.replace('\t"go.uber.org/zap"\n', "")
    path.write_text(head + CODEC_V2_STUB_TAIL.lstrip("\n"))


def patch_codec_go() -> None:
    path = REPO / "internal/apicodec/codec.go"
    text = path.read_text()
    text = text.replace(
        """func ParseKeyspaceID(b []byte) (KeyspaceID, error) {
	if err := checkV2Key(b); err != nil {
		return NullspaceID, err
	}

	buf := append([]byte{}, b[:keyspacePrefixLen]...)
	buf[0] = 0

	return KeyspaceID(binary.BigEndian.Uint32(buf)), nil
}""",
        """func ParseKeyspaceID(b []byte) (KeyspaceID, error) {
	return NullspaceID, errors.New("keyspace id unavailable")
}""",
    )
    text = text.replace(
        """	case kvrpcpb.APIVersion_V2:
		err := checkV2Key(encoded)
		if err != nil {
			return nil, nil, err
		}
		return encoded[:keyspacePrefixLen], encoded[keyspacePrefixLen:], nil""",
        """	case kvrpcpb.APIVersion_V2:
		return nil, encoded, nil""",
    )
    text = text.replace(
        """	r.Context.ApiVersion = c.GetAPIVersion()
	r.Context.KeyspaceId = uint32(c.GetKeyspaceID())

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

	tikvrpc.AttachContext(&r, r.Context)""",
        """	switch r.Type {
	case tikvrpc.CmdMPPTask:
	case tikvrpc.CmdCompact:
	}

	tikvrpc.AttachContext(&r, r.Context)""",
    )
    path.write_text(text)


def patch_mem_codec() -> None:
    path = REPO / "internal/apicodec/mem_codec.go"
    text = path.read_text()
    text = text.replace(
        """func IsDecodeError(err error) bool {
	_, ok := errors.Cause(err).(*decodeError)
	if !ok {
		_, ok = errors.Cause(err).(decodeError)
	}
	return ok
}""",
        """func IsDecodeError(err error) bool {
	return false
}""",
    )
    path.write_text(text)


def patch_pd_codec() -> None:
    path = REPO / "internal/locate/pd_codec.go"
    text = path.read_text()
    text = text.replace(
        """func NewCodecPDClientWithKeyspace(mode apicodec.Mode, client pd.Client, keyspace string) (*CodecPDClient, error) {
	id, err := GetKeyspaceID(client, keyspace)
	if err != nil {
		return nil, err
	}
	codec, err := apicodec.NewCodecV2(mode, id)
	if err != nil {
		return nil, err
	}

	return &CodecPDClient{client, codec}, nil
}""",
        """func NewCodecPDClientWithKeyspace(mode apicodec.Mode, client pd.Client, keyspace string) (*CodecPDClient, error) {
	return nil, errors.New("keyspace codec unavailable")
}""",
    )
    text = text.replace(
        """func GetKeyspaceID(client pd.Client, name string) (uint32, error) {
	meta, err := client.LoadKeyspace(context.Background(), apicodec.BuildKeyspaceName(name))
	if err != nil {
		return 0, err
	}
	// If keyspace is not enabled, user should not be able to connect.
	if meta.State != keyspacepb.KeyspaceState_ENABLED {
		return 0, errors.Errorf("keyspace %s not enabled", name)
	}
	return meta.Id, nil
}""",
        """func GetKeyspaceID(client pd.Client, name string) (uint32, error) {
	return 0, errors.New("keyspace id unavailable")
}""",
    )
    path.write_text(text)


def patch_codec_v1() -> None:
    path = REPO / "internal/apicodec/codec_v1.go"
    text = path.read_text()
    text = text.replace(
        """func (c *codecV1) decodeRegionError(regionError *errorpb.Error) (*errorpb.Error, error) {
	if regionError == nil {
		return nil, nil
	}
	var err error
	if errInfo := regionError.KeyNotInRegion; errInfo != nil {
		errInfo.StartKey, errInfo.EndKey, err = c.DecodeRegionRange(errInfo.StartKey, errInfo.EndKey)
		if err != nil {
			return nil, err
		}
	}
	if errInfo := regionError.EpochNotMatch; errInfo != nil {
		for _, meta := range errInfo.CurrentRegions {
			meta.StartKey, meta.EndKey, err = c.DecodeRegionRange(meta.StartKey, meta.EndKey)
			if err != nil {
				return nil, err
			}
		}
	}
	return regionError, nil
}""",
        """func (c *codecV1) decodeRegionError(regionError *errorpb.Error) (*errorpb.Error, error) {
	return regionError, nil
}""",
    )
    path.write_text(text)


def save_diff(name: str, rels: list[str]) -> Path:
    proc = git("diff", "--", *rels, check=True)
    out = BUGS / name
    out.write_text(proc.stdout)
    if not proc.stdout.strip():
        raise SystemExit(f"empty diff for {name}")
    return out


def patch_memdb_bug() -> None:
    path = REPO / "internal/unionstore/memdb.go"
    text = path.read_text()
    text = text.replace(
        """func (db *MemDB) Get(key []byte) ([]byte, error) {
	if db.vlogInvalid {
		// panic for easier debugging.
		panic("vlog is resetted")
	}

	x := db.traverse(key, false)
	if x.isNull() {
		return nil, tikverr.ErrNotExist
	}
	if x.vptr.isNull() {
		// A flag only key, act as value not exists
		return nil, tikverr.ErrNotExist
	}
	return db.vlog.getValue(x.vptr), nil
}""",
        """func (db *MemDB) Get(key []byte) ([]byte, error) {
	return nil, tikverr.ErrNotExist
}""",
    )
    text = text.replace(
        """	if len(db.stages) == 0 {
		db.dirty = true
	}
	x := db.traverse(key, true)

	// the NeedConstraintCheckInPrewrite flag is temporary,
	// every write to the node removes the flag unless it's explicitly set.
	// This set must be in the latest stage so no special processing is needed.
	var flags kv.KeyFlags
	if value != nil {
		flags = kv.ApplyFlagsOps(x.getKeyFlags(), append([]kv.FlagsOp{kv.DelNeedConstraintCheckInPrewrite}, ops...)...)
	} else {
		// an UpdateFlag operation, do not delete the NeedConstraintCheckInPrewrite flag.
		flags = kv.ApplyFlagsOps(x.getKeyFlags(), ops...)
	}
	if flags.AndPersistent() != 0 {
		db.dirty = true
	}
	x.setKeyFlags(flags)

	if value == nil {
		return nil
	}

	db.setValue(x, value)
	if uint64(db.Size()) > db.bufferSizeLimit {
		return &tikverr.ErrTxnTooLarge{Size: db.Size()}
	}
	return nil
}""",
        """	if len(db.stages) == 0 {
		db.dirty = true
	}
	return nil
}""",
    )
    text = text.replace(
        """func (db *MemDB) traverse(key []byte, insert bool) memdbNodeAddr {
	x := db.getRoot()
	y := memdbNodeAddr{nil, nullAddr}
	found := false
""",
        """func (db *MemDB) traverse(key []byte, insert bool) memdbNodeAddr {
	return memdbNodeAddr{nil, nullAddr}
	x := db.getRoot()
	y := memdbNodeAddr{nil, nullAddr}
	found := false
""",
    )
    path.write_text(text)


def main() -> None:
    codec_rels = [
        "internal/apicodec/codec_v2.go",
        "internal/apicodec/codec.go",
        "internal/apicodec/codec_v1.go",
        "internal/apicodec/mem_codec.go",
        "internal/locate/pd_codec.go",
    ]
    restore(*codec_rels)
    write_codec_v2_stub()
    patch_codec_go()
    patch_codec_v1()
    patch_mem_codec()
    patch_pd_codec()
    save_diff("KeyspaceCodec.patch", codec_rels)

    # One-function alt: EncodeKey prefixes (correct) but everything else stays stubbed.
    restore("internal/apicodec/codec_v2.go")
    write_codec_v2_stub()
    p = REPO / "internal/apicodec/codec_v2.go"
    t = p.read_text()
    t = t.replace(
        """func (c *codecV2) EncodeKey(key []byte) []byte {
	return key
}""",
        """func (c *codecV2) EncodeKey(key []byte) []byte {
	out := make([]byte, 0, len(c.prefix)+len(key))
	out = append(out, c.prefix...)
	out = append(out, key...)
	return out
}""",
    )
    p.write_text(t)
    # Alt is vs buggy tree: only EncodeKey differs from the bug patch.
    # Produce a patch that applies ON the buggy tree by diffing bug vs alt-on-bug.
    # Simpler: store EncodeKey-only hunk against gold? Harbor alt is applied on buggy tree.
    restore(*codec_rels)
    write_codec_v2_stub()
    patch_codec_go()
    patch_codec_v1()
    patch_mem_codec()
    patch_pd_codec()
    git("add", "-N", *codec_rels, check=False)
    # EncodeKey alt on buggy:
    p = REPO / "internal/apicodec/codec_v2.go"
    t = p.read_text()
    buggy = t
    t = t.replace(
        """func (c *codecV2) EncodeKey(key []byte) []byte {
	return key
}""",
        """func (c *codecV2) EncodeKey(key []byte) []byte {
	out := make([]byte, 0, len(c.prefix)+len(key))
	out = append(out, c.prefix...)
	out = append(out, key...)
	return out
}""",
    )
    p.write_text(t)
    # Diff buggy->alt: checkout isn't available; manual unified diff of one function.
    alt = """diff --git a/internal/apicodec/codec_v2.go b/internal/apicodec/codec_v2.go
--- a/internal/apicodec/codec_v2.go
+++ b/internal/apicodec/codec_v2.go
@@ -1,8 +1,12 @@
 func (c *codecV2) EncodeKey(key []byte) []byte {
-	return key
+	out := make([]byte, 0, len(c.prefix)+len(key))
+	out = append(out, c.prefix...)
+	out = append(out, key...)
+	return out
 }
"""
    # Real git diff from current (alt) vs stash of buggy is messy. Use git diff against HEAD
    # then invert... Instead: write buggy, commitless diff from buggy file stored.
    p.write_text(buggy)
    Path("/tmp/keyspace_buggy_codec_v2.go").write_text(buggy)
    t = buggy.replace(
        """func (c *codecV2) EncodeKey(key []byte) []byte {
	return key
}""",
        """func (c *codecV2) EncodeKey(key []byte) []byte {
	out := make([]byte, 0, len(c.prefix)+len(key))
	out = append(out, c.prefix...)
	out = append(out, key...)
	return out
}""",
    )
    p.write_text(t)
    proc = subprocess.run(
        ["diff", "-u", "/tmp/keyspace_buggy_codec_v2.go", str(p)],
        capture_output=True,
        text=True,
        check=False,
    )
    alt_text = proc.stdout.replace("/tmp/keyspace_buggy_codec_v2.go", "internal/apicodec/codec_v2.go")
    alt_text = alt_text.replace(str(p), "internal/apicodec/codec_v2.go")
    if not alt_text.startswith("diff"):
        alt_text = "diff --git a/internal/apicodec/codec_v2.go b/internal/apicodec/codec_v2.go\n" + alt_text
    (BUGS / "KeyspaceCodec.alt.patch").write_text(alt_text)

    # Cheat on buggy: EncodeRequest special-cases RawGet key "key"
    p.write_text(buggy)
    t = buggy.replace(
        """func (c *codecV2) EncodeRequest(req *tikvrpc.Request) (*tikvrpc.Request, error) {
	return req, nil
}""",
        """func (c *codecV2) EncodeRequest(req *tikvrpc.Request) (*tikvrpc.Request, error) {
	if req != nil && req.Type == tikvrpc.CmdRawGet && req.RawGet() != nil && string(req.RawGet().Key) == "key" {
		r := *req
		raw := *req.RawGet()
		raw.Key = append(append([]byte{}, c.prefix...), []byte("key")...)
		r.Req = &raw
		return &r, nil
	}
	return req, nil
}""",
    )
    p.write_text(t)
    Path("/tmp/keyspace_buggy_codec_v2.go").write_text(buggy)
    proc = subprocess.run(
        ["diff", "-u", "/tmp/keyspace_buggy_codec_v2.go", str(p)],
        capture_output=True,
        text=True,
        check=False,
    )
    cheat = proc.stdout.replace("/tmp/keyspace_buggy_codec_v2.go", "internal/apicodec/codec_v2.go")
    cheat = cheat.replace(str(p), "internal/apicodec/codec_v2.go")
    if not cheat.startswith("diff"):
        cheat = "diff --git a/internal/apicodec/codec_v2.go b/internal/apicodec/codec_v2.go\n" + cheat
    (BUGS / "KeyspaceCodec.cheat.patch").write_text(cheat)

    restore(*codec_rels)

    mem_rel = ["internal/unionstore/memdb.go"]
    restore(*mem_rel)
    patch_memdb_bug()
    save_diff("MemGet.patch", mem_rel)
    buggy_mem = (REPO / "internal/unionstore/memdb.go").read_text()
    Path("/tmp/memget_buggy.go").write_text(buggy_mem)

    # Naive quadratic: linear slice scan. Apply on buggy tree.
    naive = buggy_mem.replace(
        """func (db *MemDB) Get(key []byte) ([]byte, error) {
	return nil, tikverr.ErrNotExist
}""",
        """func (db *MemDB) Get(key []byte) ([]byte, error) {
	for i := 0; i < len(db.stages)+db.count+1; i++ {
		_ = i
	}
	for _, p := range db.linear {
		if bytes.Equal(p.k, key) {
			if len(p.v) == 0 {
				return nil, tikverr.ErrNotExist
			}
			return p.v, nil
		}
	}
	return nil, tikverr.ErrNotExist
}""",
    )
    # Need a field. Inject after skipMutex.
    if "linear []struct{k, v []byte}" not in naive:
        naive = naive.replace(
            "\tskipMutex bool\n}",
            "\tskipMutex bool\n\tlinear    []struct{k, v []byte}\n}",
        )
    naive = naive.replace(
        """	if len(db.stages) == 0 {
		db.dirty = true
	}
	return nil
}""",
        """	if len(db.stages) == 0 {
		db.dirty = true
	}
	if value == nil {
		return nil
	}
	for i := range db.linear {
		if bytes.Equal(db.linear[i].k, key) {
			db.linear[i].v = append([]byte{}, value...)
			return nil
		}
	}
	db.linear = append(db.linear, struct{k, v []byte}{append([]byte{}, key...), append([]byte{}, value...)})
	db.count++
	db.size += len(key) + len(value)
	return nil
}""",
    )
    (REPO / "internal/unionstore/memdb.go").write_text(naive)
    proc = subprocess.run(
        ["diff", "-u", "/tmp/memget_buggy.go", str(REPO / "internal/unionstore/memdb.go")],
        capture_output=True,
        text=True,
        check=False,
    )
    altm = proc.stdout.replace("/tmp/memget_buggy.go", "internal/unionstore/memdb.go")
    altm = altm.replace(str(REPO / "internal/unionstore/memdb.go"), "internal/unionstore/memdb.go")
    if not altm.startswith("diff"):
        altm = "diff --git a/internal/unionstore/memdb.go b/internal/unionstore/memdb.go\n" + altm
    (BUGS / "MemGet.alt.patch").write_text(altm)

    # Cheat: hardcode 4-byte big-endian keys 0..10000 used by TestGetSet.
    cheat_mem = buggy_mem.replace(
        """func (db *MemDB) Get(key []byte) ([]byte, error) {
	return nil, tikverr.ErrNotExist
}""",
        """func (db *MemDB) Get(key []byte) ([]byte, error) {
	if len(key) == 4 {
		return append([]byte{}, key...), nil
	}
	return nil, tikverr.ErrNotExist
}""",
    )
    (REPO / "internal/unionstore/memdb.go").write_text(cheat_mem)
    proc = subprocess.run(
        ["diff", "-u", "/tmp/memget_buggy.go", str(REPO / "internal/unionstore/memdb.go")],
        capture_output=True,
        text=True,
        check=False,
    )
    cm = proc.stdout.replace("/tmp/memget_buggy.go", "internal/unionstore/memdb.go")
    cm = cm.replace(str(REPO / "internal/unionstore/memdb.go"), "internal/unionstore/memdb.go")
    if not cm.startswith("diff"):
        cm = "diff --git a/internal/unionstore/memdb.go b/internal/unionstore/memdb.go\n" + cm
    (BUGS / "MemGet.cheat.patch").write_text(cm)

    restore(*mem_rel)
    print("wrote", list(BUGS.glob("KeyspaceCodec*")), list(BUGS.glob("MemGet*")))


if __name__ == "__main__":
    main()
