# Details — binio

1. Scalar `Read`/`Write` apply BigEndian encoding to each argument in order — the first
   failure stops the sequence mid-way. Inferable: doc.
2. The Git VLQ is the OFFSET variant, not plain VLQ: on each continuation byte the
   accumulated value is INCREMENTED before shifting and adding the next 7 bits — this is
   what removes redundant encodings. Inferable: doc — the comment spells out the scheme
   and the reference C loop.
3. Write-side mirrors it: the low 7 bits go first with no continuation bit, then each
   higher group is PREPENDED as `0x80 | (group)` after a pre-decrement — so encoding 128
   yields `0x80 0x00`... the byte pairs land big-group-first. Inferable: doc.
4. Zero encodes as one `0x00` byte — the loop never runs. Inferable: partially.
5. A NEGATIVE input to the writer never terminates — the arithmetic right-shift keeps it
   non-zero forever; there is no sign guard. Inferable: no.
6. The reader bounds the accumulator BEFORE increment+shift and fails with the overflow
   error rather than wrapping. Inferable: partially — the error var is visible.
7. `ReadUntil` drops the delimiter from the result, takes a buffered fast path when the
   reader is already buffered, and on end-of-input before the delimiter returns NIL plus
   the error — the bytes collected are discarded, not returned. Inferable: no.
8. The buffered variant returns `nil, err` whenever the read reports ANY error — even when
   it yielded a complete delimiter-terminated value at end-of-input — so the value is
   lost on EOF. Inferable: no.
9. Binary detection reads at most the sniff-window size, stops at the FIRST NUL without
   draining the reader, and reports false on short clean input — the reader is left
   positioned after the sniff window, not rewound. Inferable: doc — the comment states
   the 8000-byte window and early-exit.
10. A reader returning `(0, nil)` mid-stream is treated as end-of-data, not spun on.
    Inferable: no.
11. The sniff buffer is pooled and reused — its content beyond the bytes just read is
    stale but never inspected. Inferable: partially — the pool is visible.
12. `ReadUntil` and the object-header reader look identical but differ: this one has no
    length bound — a missing delimiter reads to EOF, never fails on size. Inferable:
    partially.
