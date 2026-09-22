# Contract — msgrecord

On-disk message-record decode and size accounting for the file store.
Every commitment below is covered by a hidden test; every hidden test
maps to a commitment.

## Commitments

1. **Record layout.** A record is a 22-byte header (length with header
   flag, sequence with erase flag, timestamp, subject length) then
   subject, optional header-length+header block, payload, and checksum;
   a well-formed record decodes all fields and matches the size
   formula. Covered by `TestDetail01`.
2. **Sanity gate.** A record whose declared length is under the header
   size, beyond the buffer, over the 32MB threshold, or whose subject
   length overruns the record is rejected as a bad message. Covered by
   `TestDetail02`.
3. **Checksum.** When a digest is supplied the trailing checksum is
   verified over the header sequence/timestamp span plus subject and
   payload; a mismatch is a bad-message error identifying the checksum;
   a nil digest skips verification. Covered by `TestDetail03`.
4. **Erased records.** A sequence carrying the erase bit decodes with
   sequence zero while the timestamp still decodes. Covered by
   `TestDetail04`.
5. **Headered slicing.** For headered records the header slice is
   capacity-limited and the message is the remainder; the borrowed
   region covers header plus payload. Covered by `TestDetail05`.
6. **Copy contract.** The no-copy decode aliases the record buffer; the
   copying decode detaches; a supplied message struct is reused.
   Covered by `TestDetail06`.
7. **Size accounting.** The raw record size is header + subject +
   payload + checksum, plus a header-length word and header bytes when
   headers exist; the too-large gate rejects a length carrying the
   header bit or exceeding the threshold. Covered by `TestDetail07`.
8. **Error shape.** The bad-message error prints the record file's
   basename plus an optional colon-separated detail. Covered by
   `TestDetail08`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — field decode and size consistency; constant values are visible |
| TestDetail02 | 2 | partially — each committed gate yields an errBadMsg |
| TestDetail03 | 3 | partially — mismatch→checksum-named error; nil digest skips |
| TestDetail04 | 4 | partially — seq=0 with ts preserved |
| TestDetail05 | 5 | partially — capacity limit and slice contents |
| TestDetail06 | 6 | yes — alias vs detach asserted via mutation |
| TestDetail07 | 7 | partially — formula and gate; estimate only asserted ≥ raw |
| TestDetail08 | 8 | no — shape only: basename plus optional detail |
