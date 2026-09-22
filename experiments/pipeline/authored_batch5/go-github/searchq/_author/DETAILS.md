# DETAILS — searchq

1. `opts.TextMatch == true` adds the text-match media type to the
   Accept header. Inferable: doc — the option exists for this; the
   media-type literal is arbitrary (assert a text-match value is
   present, not the exact string).
2. Each search type contributes its preview media type to Accept
   (commits/topics/repositories/issues). Inferable: no — which preview
   maps to which type is an arbitrary table; assert Accept is assembled
   from media types, not the specific names.
3. `parameters.RepositoryID` (when non-nil) becomes the
   `repository_id` query param. Inferable: partially — flattening a
   field to a param is derivable.
4. Accept values are joined into a single comma-separated header.
   Inferable: partially — the join is conventional.
