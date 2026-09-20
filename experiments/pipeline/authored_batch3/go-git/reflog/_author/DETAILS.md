# Details — reflog

1. A line is `<old-hash> <new-hash> <name> <<email>> <unix-secs> <±HHMM>` then, OPTIONALLY,
   a tab and the message — a line with no tab at all still parses, with an empty message.
   Inferable: doc — the format comment shows the layout; the optional tab is visible from it.
2. Blank lines in the stream are skipped silently — they end neither the stream nor produce
   an error — and a final line with no trailing newline is still parsed. Inferable: no.
3. Reading ALL entries returns the entries collected so far TOGETHER WITH the first error —
   a partial slice, not nil-with-error. Inferable: no.
4. Both hashes must satisfy the hex-hash predicate — arbitrary tokens are rejected — but the
   rest of the line keeps its field positions regardless of name contents. Inferable: partially.
5. The signature field is split at the LAST `<` and LAST `>` in the line — so a `>` inside the
   angle section is absorbed into the email, and angle brackets in the name corrupt the parse
   rather than being escaped. Inferable: no.
6. The timestamp field must be exactly two whitespace-separated fields; the seconds field is
   rejected on length (>64 chars) BEFORE numeric parsing. Inferable: no.
7. The timezone must be exactly 5 chars starting `+` or `-`; the hour/minute digits are NOT
   range-checked — `+2460` parses to an offset of 1500 minutes. Inferable: no.
8. Encoding re-derives the timezone text from the timestamp's own zone offset, producing
   `±HHMM` — a zone with sub-minute remainder silently loses the seconds. Inferable: no.
9. Before encoding, the message is normalized like canonical git: newlines and carriage
   returns become spaces, all whitespace runs collapse to one space, ends are trimmed.
   Inferable: doc — the normalize comment states the rule and cites git.
10. A message that normalizes to empty is encoded with NO trailing tab — the line ends right
    after the timezone. Inferable: partially.
11. Errors quote offending fields back, truncated at 64 bytes plus an ellipsis — a malformed
    giant line cannot blow up the error. Inferable: doc — the const's comment explains why.
12. A nil reader/writer/entry argument is an explicit error, not a panic. Inferable: partially.
