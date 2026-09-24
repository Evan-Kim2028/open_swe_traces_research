# DETAILS — markdown

1. A non-empty `opts.Mode` is sent as the request's `mode` field.
   Inferable: doc — the option exists for this.
2. A non-empty `opts.Context` is sent as the request's `context` field.
   Inferable: doc.
3. Empty option fields are omitted (not sent as empty strings), and
   `opts == nil` sends only `text`. Inferable: partially — omit-empty is
   conventional; a solver could equally send empty fields.
4. The rendered HTML body is returned as a string. Inferable: yes.
