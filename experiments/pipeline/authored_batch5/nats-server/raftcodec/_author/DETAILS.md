# Details — raftcodec

1. `appendEntry.encode` layout: `leader[idLen]` then u64 term, commit,
   pterm, pindex, then u16 entry count, then per entry `u32(1+len(data))`,
   one type byte, raw data; a uvarint `lterm` is appended AFTER the entry
   section (older decoders ignore it). Inferable: doc — the trailing-lterm
   compat note is in a source comment.
2. encode rejects: leader length neither 0 nor `idLen` → `errLeaderLen`;
   >65535 entries → `errTooManyEntries`; any entry data > MaxInt32 →
   `errBadAppendEntry`. Inferable: yes — error vars are visible.
3. encode reuses the caller's buffer when `cap(b)` suffices (slicing to
   `idLen` then appending) and allocates exactly `tlen` otherwise; returned
   length is the exact encoded size. Inferable: partially — reuse is an
   optimization visible in callers' pooled usage.
4. `decodeAppendEntry` requires `len(msg) >= appendEntryBaseLen`; per entry
   it requires 4 length bytes, `ml > 0`, and `ri+ml <= len(msg)` — anything
   else → `errBadAppendEntry`. Entry `Data` is a SLICE of `msg` (not
   copied) and `ae.buf = msg`. Inferable: yes — "can not be used post the
   wire level callback since we do not copy" comment.
5. Trailing bytes after the last entry are decoded as uvarint `lterm` only
   when `binary.Uvarint` reports n>0; a short/truncated tail leaves
   `lterm == 0`. Inferable: partially.
6. `appendEntryResponse` is fixed-width 25 bytes; `encode` copies the peer
   into bytes 16..24 (truncating or zero-padding), byte 24 = 1 iff success.
   `decodeAppendEntryResponse` returns nil unless `len(msg) == 25` and
   interns peer strings via the `peers` sync.Map (decoded peers may be
   pointer-shared). Inferable: partially — the sync.Map is visible; the
   interning intent is documented.
7. `peerState` wire form: u32 clusterSize, u32 peer count, `count*idLen`
   bytes of ids, optional trailing u16 `domainExt`. `decodePeerState`
   errors `errCorruptPeers` when buf < 8 or when fewer than the declared
   count of full ids fit; trailing >=2 bytes populate `domainExt`, fewer
   leave it zero-valued. Inferable: partially.
8. `voteRequest` is fixed-width 32 bytes; `decodeVoteRequest` returns nil
   unless `len(msg) == 32` and copies `reply` into `vr.reply`. Inferable:
   yes.
9. `encodeSnapshot(nil)` → nil. Layout: u64 lastTerm, u64 lastIndex, u32
   peerstate length, peerstate bytes, data bytes, then an 8-byte
   highwayhash-64 checksum of everything before it. Inferable: doc —
   checksum tail is described by the file's decode path.
10. `termAndIndexFromSnapFile` uses `filepath.Base`, Sscanf's
    `snap.<term>.<index>`, then REJECTS unless `fmt.Sprintf(snapFileT,
    term, index)` reproduces the basename exactly — so "snap.01.2",
    "snap.1.2x", "snap.1.2.extra", or "" all fail with `errBadSnapName`.
    Inferable: partially — strict round-trip is an intentional choice.
