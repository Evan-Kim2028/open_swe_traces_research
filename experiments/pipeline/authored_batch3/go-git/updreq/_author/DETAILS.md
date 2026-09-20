# Details — updreq

1. Command action classification: old and new ids both zero ⇒ invalid; only old zero ⇒ create;
   only new zero ⇒ delete; both non-zero ⇒ update. The zero check is on the id's bytes, not on
   which hash format the id carries. `TestCommandAction*`. Inferable: partially.
2. A request with no commands is rejected by both directions before any wire I/O
   ("commands cannot be empty"). `TestUpdReq*Suite` empty cases. Inferable: doc — the error var
   is in the kept source, but which paths raise it is not stated anywhere visible.
3. Decode accepts `shallow <id>` lines before the first command line; the id must be exactly 40
   or 64 hex digits and must be hex — anything else is a malformed request. Inferable: partially.
4. Shallow lines followed immediately by a flush and NO command lines decode as a valid empty
   request (shallow-only no-op push); a bare flush with no shallows is malformed instead.
   Inferable: no — two different empty-body outcomes that nothing in the tree distinguishes.
5. The FIRST command line carries the negotiated capability list after a NUL byte; a first line
   without the NUL separator is rejected. Later command lines have no capability field.
   Inferable: partially — sibling codecs in the same package use caps-after-NUL, visible in the
   tree, but that this message does it on exactly the first command is not stated.
6. Command grammar is `<old-id> SP <new-id> SP <name>`; the name is the complete remainder of the
   line, including any interior spaces — it is not truncated at a later space.
   Inferable: partially.
7. Object ids on commands and shallows accept 40 OR 64 hex; other lengths or non-hex characters
   are rejected as invalid. Inferable: partially — the two hex sizes are visible as kept
   constants, but that both are accepted everywhere is not obvious.
8. Decode requires a terminating flush; hitting EOF before it fails, and trailing payload after
   the flush is an error too. Inferable: partially.
9. Later command lines strip only the trailing newline — leading/interior whitespace in the name
   is preserved, not trimmed or collapsed. Inferable: no.
10. Encode order: all `shallow <id>` lines first, then the first command as
    `<old> SP <new> SP <name> NUL <caps>`, then remaining commands as bare command lines, then one
    flush. Inferable: partially.
11. Encode quirk: when the capability list is non-empty it is written after the NUL prefixed by a
    single space (`\x00` + ` ` + caps). Inferable: no — the obvious impl writes caps directly
    after the NUL.
12. On encode, the empty-commands / both-zero-ids validation runs before any bytes are emitted —
    an invalid request produces no output at all. Inferable: partially.
