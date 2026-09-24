# Details — fetchmsg

1. Request encode emits want, have and shallow hash lines each SORTED by hash —
   the caller's slice order is ignored. Inferable: no.
2. Request encode refuses to write anything when the want list is empty — the
   error is produced before the first line goes out. Inferable: no.
3. The optional request flags emit in a fixed sequence — done, thin-pack,
   no-progress, include-tag, ofs-delta, then shallows, deepen, deepen-relative,
   deepen-since, deepen-not, filter, wait-for-done — never reordered by value.
   Inferable: partially — the doc lists fields but not wire order.
4. `deepen-relative` always emits as a bare flag with no argument; the depth
   rides on the `deepen <n>` line. Inferable: partially — the flag shape is
   the natural reading, the no-argument form is the detail.
5. `deepen-since` encodes as UTC unix seconds. Inferable: partially.
6. Request decode stops cleanly on a flush-pkt, a delim-pkt, end-of-input, OR
   a blank line — all four end the argument list without error. Inferable:
   partially — the doc says "until a flush-pkt" only.
7. Request decode silently ignores lines it does not recognize — unknown
   keywords are skipped, not rejected. Inferable: no.
8. A `deepen-relative` line carrying an argument is still accepted — the
   argument is ignored and only the flag is set. Inferable: no.
9. Hash fields on want/have/shallow lines are validated as full-length object
   IDs; a malformed value fails the decode. Inferable: partially — the kept
   hash helper reveals full-length parsing, the rejection is the detail.
10. Each list-like section on decode is capped by the kept `maxSectionLines`
    bound; exceeding it fails rather than growing unbounded. Inferable: doc —
    the var and its rationale comment are visible.
11. Response decode enforces the fixed section order
    (acknowledgments < shallow-info < wanted-refs < packfile-uris < packfile):
    a repeated or out-of-order header is a malformed-response error, not a
    reorder. Inferable: doc — the grammar comment shows the order.
12. A `ready` line inside acknowledgments commits the response to the
    packfile shape: it must be followed by a delim-pkt, and a premature
    flush/response-end/end-of-input afterwards is a malformed-response error.
    Without `ready`, the section must end the whole response. Inferable:
    partially — the two shapes are documented; the strict pairing is not.
13. Reaching the `packfile` section header sets the flag and returns with the
    reader positioned at the first packfile pkt-line — decode never consumes
    pack bytes. Inferable: doc — stated in the doc comment.
14. Response encode of acknowledgments writes ACK lines first, then a single
    `ready` when set — and emits `NAK` only when there were no ACKs and no
    ready. Inferable: doc — the grammar comment spells it out.
15. Response encode with no packfile writes acknowledgments terminated by a
    flush-pkt and rejects the shape when acknowledgments are missing, marked
    ready, or accompanied by metadata sections. Inferable: partially.
16. The packfile-uris section decode trims only the trailing newline — other
    whitespace inside a URI line is preserved, unlike the space-trimming used
    by sibling sections. Inferable: no.
