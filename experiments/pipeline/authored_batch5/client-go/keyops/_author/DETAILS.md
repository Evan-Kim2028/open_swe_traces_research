# Details — keyops

1. `NextKey(k)` returns `k` with one `0x00` appended — the immediate
   successor in byte order. Inferable: doc.
2. `PrefixNextKey(k)` increments the LAST byte and propagates carry left:
   `"a\xff"` → `"b"`, `"ab"` → `"ac"`. It never extends the slice. Inferable:
   doc — the seek example is in the comment.
3. An all-`0xFF` key (carry out of the top byte) returns an EMPTY slice, not
   `{0xFF,0x00}` — the comment calls this out as a deliberate difference
   from upstream. Inferable: doc.
4. `PrefixNextKey` of an empty key returns an empty slice (no bytes to bump,
   loop trips the all-carry exit). Inferable: no — a corner of the loop.
5. `CmpKey` is lexicographic byte order: -1/0/+1. Inferable: doc.
6. `StrKey` renders the key as lowercase hex (`[]byte{0xde,0xad}` →
   `"dead"`). Inferable: partially — hex is conventional, case is a choice.
7. `ReplicaReadType.IsFollowerRead` is true for every value EXCEPT
   `ReplicaReadLeader`. Inferable: partially — "not leader" is the obvious
   reading but follower-vs-mixed-vs-learner inclusion is a choice.
8. `ReplicaReadType.String` maps leader/follower/mixed/learner/
   prefer-leader to their lowercase names and any other byte to
   `unknown-<n>`. Inferable: partially — the unknown-format string is a
   choice.
