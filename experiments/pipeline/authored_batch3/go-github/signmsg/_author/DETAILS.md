# DETAILS — signmsg

1. A nil signer is rejected with an error. Inferable: yes — defensive
   boundary, forced by the signature.
2. The payload is `tree <sha>`, then one `parent <sha>` per parent, then
   `author <name> <<email>> <unix> <±HHMM>`, then the same shape for
   `committer` followed by a blank line, then the commit message — all
   newline-joined. Inferable: partially — the Git object format is
   documented convention, exact spacing is arbitrary.
3. A nil committer falls back to the author. Inferable: doc — the
   createCommit doc implies committer defaults to author.
4. A nil commit, nil/empty message, or nil author is rejected before any
   serialization. Inferable: partially — the field set is arbitrary.
5. The signature is whatever `signer.Sign` writes for that payload —
   streamed, not buffered per line. Inferable: yes — the MessageSigner
   interface forces it.
