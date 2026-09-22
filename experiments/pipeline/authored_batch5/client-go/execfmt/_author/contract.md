# Contract (L2) — execfmt

`FormatDuration` returns the duration's own string at or below one
microsecond and otherwise truncates to the largest unit, keeping two
decimals below ten units and one at or above, so that rounding may carry
into the next unit's display. `MergeFromTimeDetail` prefers the V2 detail
and reads its nanosecond fields as nanoseconds — including the suspend
field; the V1 path reads its millisecond fields as milliseconds and its
nanosecond total field as nanoseconds, and never sets suspend time.
`MergeFromScanDetailV2` is nil-safe, renames the version fields onto the
key fields, and accumulates the RocksDB counters and the two nanosecond
durations. The three `Merge` methods accumulate every field and are safe
under concurrent calls. `RUDetails` accumulates read RU, write RU, and
wait duration on update, treats a nil receiver or nil consumption as a
no-op, and snapshots on clone and merge. `MergeFromWriteDetailPb` maps
each nanosecond field to a duration and ignores a nil argument.

| test | commitment |
| --- | --- |
| `TestDetail01` | Below/at one microsecond the input string is returned; above it the documented pruning literals hold, including the rounding carry. |
| `TestDetail02` | With both details present the V2 values land on all five fields as nanosecond durations; with only V1 the millisecond fields land as milliseconds, the nanosecond total lands as nanoseconds, and suspend stays zero. |
| `TestDetail03` | Nil input is a no-op; the three version fields land on the renamed key fields; the RocksDB counters and the two durations accumulate on repeated merges. |
| `TestDetail04` | Each merge adds every field to the receiver, including under fifty concurrent merges. |
| `TestDetail05` | Update accumulates both RUs and the wait duration; nil consumption and nil receiver are no-ops; the constructor, clone, and merge carry the values. |
| `TestDetail06` | Every nanosecond field maps to the corresponding duration field; nil input is a no-op. |
