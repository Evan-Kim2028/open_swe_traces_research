# Details — bytesfmt

1. `FormatBytes(n)` delegates to `BytesToString` when `n <= 1024` OR when the
   chosen unit is Bytes — so `FormatBytes(1024)` is `"1024 Bytes"`, not
   `"1 KB"`. Inferable: no — the <= boundary is a choice.
2. `FormatBytes` picks GB/MB/KB by strict `>` thresholds; decimals: 0 when
   `n` is an exact unit multiple, else 2 when `n/unit < 10`, else 1 —
   `FormatBytes(1025)` = `"1.00 KB"`, `FormatBytes(20971521)` = `"20.0 MB"`,
   `FormatBytes(2048)` = `"2 KB"`. Inferable: no.
3. `BytesToString` uses strict `>1` float comparisons per unit and always
   prints the raw `%v` float — `BytesToString(1024)` = `"1024 Bytes"`,
   `BytesToString(2048)` = `"2 KB"`. Inferable: no.
4. `CompatibleParseGCTime` parses with `gcTimeFormatOld`
   (`20060102-15:04:05 -0700`); Go's parser accepts an optional fractional
   seconds field, so new-format values parse directly. On failure it drops
   the LAST space-separated field and reparses (handles a trailing zone
   abbreviation like `CST`); on second failure it returns an error.
   Inferable: partially — the fallback is documented, the exact error text
   is not committed.
5. `EncodeToString` hex-encodes into a `[]byte` (not string);
   `HexRegionKey` = uppercase(hex) via `ToUpperASCIIInplace`;
   `HexRegionKeyStr` wraps it as string. Inferable: doc.
6. `ToUpperASCIIInplace` mutates and returns the input, only touching
   `a`..`z`. Inferable: yes.
7. `String` is a zero-copy `[]byte -> string` view. Inferable: doc.
