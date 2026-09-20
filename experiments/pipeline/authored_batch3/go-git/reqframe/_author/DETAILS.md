# Details — reqframe

1. An empty `Command` encodes as a single flush packet — the whole "empty request", nothing
   else. Inferable: doc — the wire grammar sits in the kept doc comment.
2. A non-empty command request is `command=<name>\n`, then the capability list, then a delim
   packet, then the args body, then a flush — in that order. Inferable: doc.
3. A flush as the FIRST packet on decode leaves `Command` empty and succeeds — an empty request,
   not an error. Inferable: doc.
4. The first non-flush line must begin `command=` after its trailing newline is trimmed;
   anything else is a malformed-request failure. Inferable: partially.
5. Capabilities are read until a delim packet — any other packet kind there is a failure, and
   the command args begin only after the delim. Inferable: partially.
6. With no args decoder installed, the packet after the delim must be a flush — a data packet
   there is a failure. Inferable: partially.
7. Hitting end-of-input before the first packet decodes as an empty request (no error), not as
   a truncated request. Inferable: no.
8. The git-transport request is ONE packet line: `<command> SP <pathname> NUL [host=<host> NUL]
   [NUL <param> NUL ...]` — host emitted only when non-empty, params each NUL-terminated behind
   an extra NUL leader. Inferable: partially.
9. Both directions validate the request fields: an empty command, or ANY ASCII control byte
   (0x00-0x1f and 0x7f) in command, pathname, host or a parameter, is an invalid-request
   failure — no wire bytes are produced on encode. Inferable: doc — the field validator's doc
   comment stays visible.
10. On decode the request line must end in a NUL byte; a missing terminator, an empty line, a
    line without the command/pathname space, or a flush packet each give a different
    early-exit outcome (silent end-of-input vs invalid-request). Inferable: no.
11. On decode the field after the first NUL becomes the host value — its `host=` prefix is
    stripped when present but NOT required; a bare second field still lands in Host.
    Inferable: no.
12. Empty parameter slots on decode (from doubled terminators) are dropped, not kept as empty
    strings. Inferable: partially.
13. Decode validates the parsed fields with the same control-byte rule as encode — a peer's
    request carrying newline or ESC fails after parsing, symmetrically. Inferable: partially.
