1. Index version 4 writes no NUL padding after an entry. Inferable: partially
2. Versions other than 4 pad so that `wrote + padLen` is a multiple of 8, with `padLen = 8 - wrote%8` (so a multiple of 8 still writes 8 NULs). Inferable: no
3. v4 names are prefix-compressed against the previous entry's name (entries are already sorted). Inferable: doc
4. The encoder writes a variable-width integer equal to `len(previousName) - commonPrefixLen(previous, current)`, or 0 for the first entry. Inferable: no
5. It then writes the suffix `current[prefix:]` followed by a single NUL. Inferable: no
6. `commonPrefixLen` is a byte-wise (not rune-wise) longest shared prefix. Inferable: partially
7. `lastEntry` is updated to the current entry before returning, including on a later write error. Inferable: no
8. v2/v3 still write the name as raw bytes with no NUL of their own; padding supplies the NULs. Inferable: yes
