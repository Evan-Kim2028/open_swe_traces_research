1. If `len(path) <= len(domain)` the match fails. Inferable: yes
2. The path must start with the domain components, compared exactly. Inferable: yes
3. A pattern with a single segment (no `/` in the original string, including patterns that are only a glob) is matched only against the last component of `path`. Inferable: no
4. A multi-segment pattern is matched against `path[len(domain):]`. Inferable: partially
5. Empty pattern segments are skipped. Inferable: no
6. A `**` segment eats itself and, if it was the last segment, succeeds immediately. Inferable: partially
7. A pattern segment that contains `**` as a substring of a larger token (not a whole segment) fails the whole match. Inferable: no
8. After a `**`, later path components are scanned until `filepath.Match` hits the next pattern segment. Inferable: partially
9. Without a pending `**`, each path component must `filepath.Match` the next pattern segment in lockstep. Inferable: yes
10. If path components run out while pattern segments remain, the match fails. Trailing `**` already returned true in rule 6. Inferable: yes
11. A `filepath.Match` error (malformed char class) makes `Match` return false, not an error. Inferable: no
12. Simple patterns do not match a component in the middle of the path, only the last one. Inferable: no
