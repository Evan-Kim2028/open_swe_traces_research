1. `foldKey(a) == foldKey(b)` if and only if `strings.EqualFold(a, b)`. Inferable: doc
2. An all-ASCII name with no uppercase letters is returned unchanged and must not allocate. Inferable: doc
3. Each rune is mapped to the smallest rune in its `unicode.SimpleFold` orbit, then lowercased if that smallest rune is an ASCII letter. Inferable: no
4. ASCII uppercase is folded by adding `'a'-'A'`, not by walking the orbit. Inferable: no
5. `foldKey("Core")` equals `foldKey("core")` and equals `"core"`, not `"CORE"`. Inferable: doc
6. U+017F (long s) shares a key with `"s"` / `"S"`. `strings.ToLower` does not. Inferable: no
7. U+212A (kelvin sign) shares a key with `"k"` / `"K"`. Inferable: no
8. Invalid UTF-8 is decoded the way `strings.EqualFold` decodes it (U+FFFD), so those names still collide correctly. Inferable: no
