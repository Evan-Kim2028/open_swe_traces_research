# Contract — hdrsurgery

Header-block surgery and reply-subject classification on the inbound
message path. Every commitment below is covered by a hidden test; every
hidden test maps to a commitment.

## Commitments

1. **Key lookup.** A header key is found only when preceded by a full
   CRLF and immediately followed by a colon; a longer key that merely
   starts with the searched key does not shadow the real one, and a key
   occurring inside a value is not a match. Covered by `TestDetail01`.
2. **Value extraction.** The sliced value borrows into the header with
   capacity capped at the value end; leading spaces after the colon are
   skipped and the value ends at the first CRLF. The copied form returns
   an exact-size copy. Covered by `TestDetail02`.
3. **Exact-key removal.** Removing a key removes every exact-key line,
   not just the first; longer keys that start with the key survive; when
   only the empty header line remains the result is nil; an absent key
   leaves the header unchanged. Covered by `TestDetail03`.
4. **Prefix removal.** Lines whose key begins with the prefix are
   removed; a prefix occurrence inside a value is skipped while scanning
   continues; an absent prefix leaves the header unchanged. Covered by
   `TestDetail04`.
5. **Status removal (shape).** A status line is stripped only when the
   header begins with the protocol marker and a carriage return exists
   after it; a non-matching header is returned unchanged, and stripping
   that leaves only the empty header line yields nil. Covered by
   `TestDetail05`.
6. **Reply classifiers.** A service reply is the four-byte service
   prefix; a JetStream ack subject is the ack prefix with at least one
   byte beyond it. Covered by `TestDetail06`.
7. **Deliver marker.** The deliver-marker offset in an encoded ack reply
   is the first at-sign occurring after at least eight dots; an at-sign
   earlier than that, a reply with no later at-sign, and a non-ack reply
   all yield -1. Covered by `TestDetail07`.
8. **Reserved replies.** Service replies, JetStream ack subjects, and
   gateway routed replies (current and legacy prefixes) are reserved;
   ordinary subjects are not. Covered by `TestDetail08`.
9. **Subject/queue split.** One whitespace-separated field yields the
   subject; two yield subject and queue; zero or more than two is an
   error, as is an invalid subject or queue token. Covered by
   `TestDetail09`.
10. **Arg split.** Protocol args tokenize on space, tab, CR and LF with
    runs collapsed, no empty tokens, and more than the inline capacity
    still returns all tokens. Covered by `TestDetail10`.

## Coverage

| Hidden test | DETAILS row | Inferable |
|---|---|---|
| TestDetail01 | 1 | partially — prefix-shadowing and line-start gate only |
| TestDetail02 | 2 | doc — borrowed slice with value-capped capacity |
| TestDetail03 | 3 | partially — all-duplicates removal and nil reduction |
| TestDetail04 | 4 | partially — strict-prefix line removal and line-start gate only; whole-key-equal-prefix edge not pinned |
| TestDetail05 | 5 | no — shape only: unchanged vs stripped vs nil |
| TestDetail06 | 6 | yes |
| TestDetail07 | 7 | doc — eight-dot gate and -1 shapes |
| TestDetail08 | 8 | yes |
| TestDetail09 | 9 | yes |
| TestDetail10 | 10 | yes |
