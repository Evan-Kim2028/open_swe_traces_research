# DETAILS — pullmerge

1. When `commitMessage` is empty and `options` is nil or has
   `DontDefaultIfBlank == false`, the request body omits
   `commit_message` (nil pointer). Inferable: partially — omission is
   derivable from the option's name; the encoding as nil is
   conventional.
2. When `commitMessage` is empty and `options.DontDefaultIfBlank` is
   true, the request body carries `commit_message` as an explicit empty
   string (non-nil pointer to ""). Inferable: no — the
   flag-times-empty-message interaction is an arbitrary pairing; the
   verifier asserts only that the field is present-and-empty.
3. A non-empty `commitMessage` is sent regardless of the flag.
   Inferable: yes.
