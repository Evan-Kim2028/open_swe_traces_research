# Details — zonespec

1. A bare `example.com` spec yields `Name` set with a trailing dot applied; `ID` left empty. Inferable: yes — the dot-suffix helper is intact in the tree.
2. `*/1234` yields a spec with only `ID` set (name empty) — `*` on the left of `/` selects id-matching. Inferable: partially — the `*/` convention is a choice.
3. `name/1234` sets both `Name` and `ID`. Inferable: yes.
4. The spec splits on the FIRST `/` only — `a/b/c` gives `Name:"a."`, `ID:"b/c"`. Inferable: no — arbitrary.
5. Spec input is whitespace-trimmed before parsing. Inferable: yes.
6. Rules entries `*` and `*/*` set `Wildcard` and are not added to `Zones`. Inferable: partially.
7. An empty rules list means `Wildcard=true` (permit everything, with an informational log). Inferable: partially — the permit-all default is a choice.
8. Matching skips a rule whose `Name` is set and differs, but a rule whose `ID` is set and differs vetoes the whole match immediately (`continue` vs `return false` asymmetry). Inferable: no — the asymmetry is arbitrary.
9. The zone's name is dot-suffixed before comparison; the id is compared verbatim. Inferable: yes.
10. A wildcard `ZoneRules` never matches "explicitly" — only `Zones` entries are scanned. Inferable: partially.
