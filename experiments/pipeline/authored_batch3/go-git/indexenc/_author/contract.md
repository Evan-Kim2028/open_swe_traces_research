# Contract — indexenc

Hidden suite: `tests/hidden/plumbing/format/index/indexenc_bb_test.go`
(package `index`, in-package). One `TestDetailNN` per DETAILS.md line.
The intact sibling decoder is used as ground truth for round-trips; raw bytes
are parsed directly where the wire layout itself is the commitment.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | doc | Output opens `DIRC` + u32 version + u32 entry count; `Version = EncodeVersionSupported+1` fails with zero bytes written. |
| TestDetail02 | 2 | partially | Three entries `{b/TheirMode, a/Merged, b/AncestorMode}` round-trip in order `a/Merged, b/AncestorMode, b/TheirMode` — name sort then stage sort, caller order discarded. |
| TestDetail03 | 3 | doc | All fixed-part fields (both timestamps incl. nanoseconds, dev, ino, mode, uid, gid, size, stage, hash) round-trip identically through v3. |
| TestDetail04 | 4 | partially | A 5000-byte name encodes with flags low 12 bits = `0xFFF` (saturated) and stage in bits 12-13; the long name still round-trips. |
| TestDetail05 | 5 | partially | Intent-to-add entry carries bit `0x4000` in its flags plus a second u16 with the intent-to-add bit; a plain entry in the same index has no extended flag; both extended bits round-trip. |
| TestDetail06 | 6 | partially | Zero `time.Time` encodes without error; `time.Unix(-5,0)` fails. |
| TestDetail07 | 7 | partially | v2 entries pad (fixed+hash+name) to an 8-byte boundary with NULs — exact pad sizes checked for 3 name lengths (8, 4, 3), every pad byte NUL. |
| TestDetail08 | 8 | doc | v4 emits no padding (exact total size 234 for a 3-entry fixture); entry 2 writes varint strip count `0x02` + `"ef\x00"`; entry 3 writes `0x03` + `"xyz\x00"`; all names round-trip. |
| TestDetail09 | 9 | partially | First v4 entry carries strip count `0x00` followed by the full name + NUL. |
| TestDetail10 | 10 | no | SHAPE: an intent-to-add v3 entry's total footprint (fixed+ext+name+pad) equals `8 - (62+2+5) % 8` padding counted from the *end of the name* — i.e. the extended word is inside the padded region, verified by the trailer landing exactly at `total+pad`. |
| TestDetail11 | 11 | no | SHAPE: last 20 bytes = SHA-1 of all preceding bytes; under `WithSkipHash()` the trailer is 20 zero bytes and still decodes with the same option. |
| TestDetail12 | 12 | partially | `encodeRawExtension` rejects 7-char and 2-char signatures, accepts 4-char and writes `sig + u32 len + payload` verbatim. |

Refusals/softening: line 10's commitment ("padding counts the ext word") is asserted as a
total-footprint invariant rather than a specific pad count, because which boundary the
implementation pads from is the non-inferable part; line 11 asserts observable trailer
content, not the tee wiring.
