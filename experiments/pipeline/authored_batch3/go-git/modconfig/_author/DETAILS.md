# Details — modconfig

1. A submodule name is rejected when empty, `.`, containing a NUL byte, starting or ending
   with a `/` or `\`, or having a drive-letter `x:` prefix — and per-component, when the
   component resolves to `..` under EITHER HFS+ ignored-codepoint normalisation OR NTFS
   trailing-dot/space/ADS canonicalisation (both checks always run). Inferable: doc — the
   kept comment enumerates the intent; the exact check set is the excised body.
2. `.gitmodules` entries whose NAME or PATH fails validation are silently SKIPPED during
   unmarshal — they never reach the map — while empty path/URL errors do NOT skip.
   Inferable: no.
3. A path containing a `..` component (either separator, either end) fails validation —
   the regex is visible — but a `..` inside a component name is legal. Inferable: partially.
4. Marshaling a submodule with an empty name falls back to using its PATH as the
   subsection name. Inferable: no.
5. A branch's `remote`/`merge`/`rebase`/`description` options are REMOVED from the raw
   subsection when the field is empty — not written empty. Inferable: no.
6. Branch `rebase` accepts `true`, `interactive`, AND the undocumented `false` — anything
   else fails; `merge` values without a `refs/` prefix are accepted (kept comment).
   Inferable: doc — the field comment lists the values; the `false` acceptance is in the
   excised body.
7. Description marshals with real newlines rewritten to a literal `\n` escape (and back on
   unmarshal) — a workaround for the config encoder's quoting, per the kept hack comment.
   Inferable: doc — the comment explains the hack; whether BOTH directions transform is in
   the body.
8. `parseConfigBool` mirrors upstream maybe-bool: true/yes/on and false/no/off
   case-insensitively, plus ANY decimal integer (0 false, non-0 true) — and the empty string
   maps to UNSET, diverging from upstream's false. Inferable: doc — the comment states all
   of it including the divergence.
9. An unrecognised bool value is UNSET (platform default applies), not an error and not
   false. Inferable: doc.
10. `String` on the unset value renders `"unset"` — a third spelling distinct from the
    config spellings. Inferable: no.
11. `Modules.Marshal` preserves existing raw subsections (round-trips unknown options)
    rather than building a fresh config — only the submodule list is replaced. Inferable:
    no.
12. Submodule validation order: bad-name first (wrapped with the quoted name), then empty
    path, then empty URL, then bad path — the empty checks precede the `..` check.
    Inferable: no.
