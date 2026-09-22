# Bug report — msgrecord

**Title:** Corrupt block records decoded as valid; erased sequences
surface with garbage sequence numbers.

**Symptoms:**
- A record whose `rl` exceeds the buffer or the 32MB threshold is
  silently accepted, so `LoadMsg` returns torn messages instead of
  `errBadMsg`.
- Erased records (`ebit` set) decode with the raw stored seq instead
  of seq=0, resurrecting deleted stream entries.
- Headered messages are sliced as `subj|msg` (the 4-byte hlen field
  treated as message bytes), corrupting every message carrying NATS
  headers after restart.
- `sm.hdr` is returned without the capacity limit, so appends into a
  caller's header slice scribble over the message bytes.

**Reproduction:** write a message with headers, force a block reload
(`TestFileStoreCorruptionSetsHbitWithoutHeaders` zeroes the record
after a tombstone write), LoadMsg returns garbage rather than
`errBadMsg`.

**Expected:** strict rl/dlen/shlen bounds, checksum over the exact
documented ranges, ebit → seq 0, capacity-limited hdr slice.
