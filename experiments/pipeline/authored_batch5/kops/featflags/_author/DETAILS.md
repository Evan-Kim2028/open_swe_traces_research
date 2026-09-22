# Details — featflags

1. `Enabled` precedence: explicit `enabled` (set by `ParseFlags`) beats `defaultValue`;
   both nil → false. Inferable: yes.
2. `new` is idempotent per key — a second registration returns the existing flag and does
   NOT overwrite an existing default (first-writer-wins on `defaultValue`). Inferable:
   partially — first-wins vs last-wins is a choice.
3. `ParseFlags` accepts `+Name`/`-Name` prefixes AND bare names (bare = enable). Whitespace
   around the whole string and each item is trimmed; empty items skipped. Inferable: yes —
   the grammar is the only consistent reading.
4. Unknown flag names are logged and ignored — NOT an error. Inferable: yes (Get is the
   strict API).
5. Parse order matters — later items override earlier ones for the same flag (`A,-A` →
   disabled). Inferable: yes.
6. `init()` parses the env var once at package load; mutating the env later has no effect
   unless `ParseFlags` is called again. Inferable: doc.
7. `Get` errors `flag %s not found` on unknown names. Inferable: no — error text arbitrary.
