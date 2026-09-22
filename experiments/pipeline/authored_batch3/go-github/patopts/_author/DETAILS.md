# DETAILS — patopts

1. Each Owner entry becomes its own `owner[]=<escaped>` query pair, order
   preserved. Inferable: partially — repetition is required by the API
   shape the tests assert, the `[]` spelling is an endpoint convention.
2. Each TokenID entry becomes its own `token_id[]=<decimal>` pair. Inferable:
   partially — same convention.
3. The array params join the existing query with `&`, or start it with `?`
   when no query is present yet. Inferable: yes — URL mechanics.
4. A nil opts returns an error rather than a bare listing URL. Inferable:
   no — the choice to error instead of listing all is arbitrary.
5. Other option fields still flow through the generic addOptions encoder.
   Inferable: yes — shared mechanism.
