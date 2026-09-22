1. `IsHFSDot(part, needle)` is true iff, after dropping a private set of ignored Unicode code points from `part` and folding remaining ASCII letters to lower case, the result equals `"."+needle`. Inferable: partially
2. Ignored code points may appear before the dot, between letters of the needle, and after the last letter; they are skipped rather than matched. Inferable: no
3. The ignored set is the HFS+ ignorable list used by Git: U+200C, U+200D, U+200E, U+200F, U+202A–U+202E, U+206A–U+206F, U+FEFF. Inferable: no
4. After skipping ignored code points, the next remaining rune must be `'.'`. Inferable: yes
5. Each needle rune is then matched against the next non-ignored rune of `part`. Inferable: yes
6. A non-ignored rune above 127 in a needle position makes the match fail. Inferable: no
7. ASCII case folding uses `unicode.ToLower` on the remaining rune, which is required to be ASCII. Inferable: partially
8. After the needle is consumed, only ignored code points may remain; any other leftover rune fails. Inferable: yes
9. An empty `part`, a lone `"."`, or a needle that does not consume the whole (non-ignored) spelling fails. Inferable: yes
10. `IsHFSDotGit(".GIT")` is true; `IsHFSDotGit(".gitmodules")` is false. Inferable: yes
