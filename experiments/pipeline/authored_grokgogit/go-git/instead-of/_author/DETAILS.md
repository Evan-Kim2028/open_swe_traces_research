1. `ApplyInsteadOf` rewrites using only that one `URL`'s `InsteadOfs`. Inferable: yes
2. A candidate matches when the remote URL has that `insteadOf` string as a prefix. Inferable: yes
3. If several prefixes match, the longest prefix wins. Inferable: doc
4. Equal-length matches keep the first in config / slice order. Inferable: no
5. The rewrite is `Name + remoteURL[len(prefix):]`. Inferable: partially
6. No match returns the original URL and `matched=false` from the helper. Inferable: yes
7. Across several `*URL` values, the longest prefix still wins even if it belongs to a later URL. Inferable: no
8. A match of length 0 is treated as no match. Inferable: no
