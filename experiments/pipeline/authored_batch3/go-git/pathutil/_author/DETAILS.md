# Details — pathutil

1. `.git` matches case-insensitively AND its 8.3 alias `git~1` — but `git~2` or `GIT~9`
   are NOT covered by that predicate. Inferable: doc — the comment names exactly the two.
2. HFS+ matching skips the ignored-codepoint table between ANY expected bytes, so
   `.<U+200C>git` still matches; the needle is compared case-insensitively but a
   non-ASCII byte never equals a needle rune; the component must END after the needle —
   `.gitx` doesn't match. Inferable: partially — the ignored table is visible.
3. NTFS pattern 1 is `.<needle>` followed only by spaces/periods — a `:` ends the scan
   early (ADS suffix always allowed), so `.git::$INDEX_ALLOCATION` matches. Inferable:
   partially.
4. NTFS pattern 2 accepts `<first-6-of-needle>~` followed by digit ONE THROUGH FOUR plus
   trailing junk — `gitmod~4` is caught, `~5` is not. Inferable: no.
5. The reserved-name check compares only the base: `CON`, `con.txt`, `NUL:x`, `LPT3 `
   all match — a name merely PREFIXED by a reserved word without a space/dot/colon
   follower does not. Inferable: partially — the name list is visible.
6. `WindowsValidPath` treats a plain `.git` or `git~1` as VALID — only the NTFS-disguised
   variants fail, plus every reserved device name. Inferable: no.
7. `ValidTreePath` rejects control bytes, empty/separator-only paths, `.` and `..`
   components, volume-name prefixes, and every `.git` disguise at ANY component position
   — but deliberately does NOT reject Windows device names. Inferable: doc — the comment
   enumerates all of this.
8. `HasUnsafeComponent` scans both `/` and `\`-separated components for control bytes,
   then runs the fold checks ONLY on components containing a `.` — a control byte still
   fires anywhere. Inferable: doc — the comment explains the dot gate.
9. The unsafe-component fold uses exactly three of the four needle combinations — the
   fourth is skipped because it is strictly subsumed. Inferable: doc — the comment
   explains the omission.
10. `~` expansion: only `~/…` and `~user/…` forms expand — a bare `~` or `~x` with no
    slash is returned unchanged; a lookup failure returns the ORIGINAL path with the
    error, not an empty string. Inferable: no.
11. Backslash is a path separator for these predicates even on non-Windows hosts — the
    checks are platform-independent by design. Inferable: doc.
12. `IsNTFSDot` pattern 3 uses the caller-supplied short-name prefix — each wrapper
    carries its own baked-in prefix. Inferable: partially.
