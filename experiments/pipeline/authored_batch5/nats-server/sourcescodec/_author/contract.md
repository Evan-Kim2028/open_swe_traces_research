# Contract — sourcescodec

Stream-source tracking codec: `sources.db` persistence plus the
`Nats-Stream-Source` header parse/generate pair. Every commitment below
is covered by a hidden test; every hidden test maps to a commitment.

## Commitments

1. **File layout.** `sources.db` is a version byte 1, a little-endian
   count, and a little-endian stamp equal to last-sequence + 1; each
   source is a uvarint-length name, a uvarint sequence, and a
   uvarint-length ident. Covered by `TestDetail01`.
2. **Decode strictness.** Under the header length is a short-buffer
   error; a non-1 version is the invalid-version error; every per-entry
   field — including a declared length overrunning the buffer — is
   unexpected-EOF; the sources map is allocated only when the count is
   positive. Covered by `TestDetail02`.
3. **Header dispatch.** A header starting with the ack prefix uses the
   legacy decoder; otherwise the value is space-split and only arities
   of 2 or >= 4 are accepted. Covered by `TestDetail03`.
4. **v2 reconstruction.** The index name re-joins fields 0, 2, and 3;
   the sequence comes from field 1 via the ack-number parser; field 5
   is the ident when present; arity mismatches return empties. Covered
   by `TestDetail04`.
5. **Header generation.** The generated header is the index-name parts
   interleaved with the source sequence lifted from the inbound ack
   reply (token 5 for v1, token 7 for v2, defaulting to "1"), the
   original subject, and the ident only when non-empty. Covered by
   `TestDetail05`.
6. **Ack-reply decode.** The legacy branch extracts the stream and
   source sequence from both v1 and v2 reply layouts and returns
   empties on malformed input. Covered by `TestDetail06`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — layout + LastSeq+1 stamp asserted on the written file |
| TestDetail02 | 2 | yes — exact error identities per truncation |
| TestDetail03 | 3 | partially — arity split asserted; 2-field seq value left unpinned |
| TestDetail04 | 4 | partially — field positions asserted including ident |
| TestDetail05 | 5 | partially — both ack layouts + default + ident elision asserted |
| TestDetail06 | 6 | partially — v1/v2 extraction and malformed→empty asserted |
