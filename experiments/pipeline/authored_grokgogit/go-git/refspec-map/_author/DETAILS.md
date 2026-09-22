1. `Validate` requires exactly one `:` and a non-empty destination after it; otherwise it returns `ErrRefSpecMalformedSeparator`. Inferable: partially
2. The source and destination sides must contain the same number of `*` characters, and that number must be 0 or 1; otherwise `ErrRefSpecMalformedWildcard`. Inferable: no
3. A leading `+` is a force flag and is not part of the source. `Src` already implements this. Inferable: yes
4. A spec whose first character is `:` is a delete (empty source). Inferable: yes
5. Non-wildcard `Match` is exact equality of `Src()` and the reference name. Inferable: yes
6. Wildcard `Match` splits `Src()` on the first `*` into prefix and suffix; the name must be at least as long as prefix+suffix, start with prefix, and end with suffix. Inferable: partially
7. Non-wildcard `Dst` returns the text after `:`. Inferable: yes
8. Wildcard `Dst` copies the slice of the name that sat under the source `*` into the destination `*`. For `refs/heads/*bc` against `refs/heads/abc` that slice is `a`. Inferable: no
9. `MatchAny` is true if any spec in the list matches. Inferable: yes
10. Error values are the two package sentinels, not free-form strings. Inferable: doc
