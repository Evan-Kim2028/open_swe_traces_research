# Commitments — goname

1. SnakeCase inserts "_" before uppercase letters that follow lowercase
   or precede a lowercase (handling runs like "HTTPServer" ->
   "http_server"), lowercases everything, converts spaces, dashes, and
   slashes to underscores, and maps the "OAuth" exception to "oauth".
   In-tree coverage: deleted tests only. Inferable: no — the run-split
   rule and exception table are implementation detail.
2. KebabCase is SnakeCase with underscores replaced by dashes and a
   trailing underscore removed. In-tree coverage: deleted tests only.
   Inferable: partially.
3. Goify strips non letter/digit characters, uppercases the first letter
   of each word when firstUpper, guarantees a letter or "_" start, and
   honors Go reserved keywords via fixReservedGo. In-tree coverage:
   deleted tests only. Inferable: partially — the keyword table is not
   spelled out.
4. GoifyAtt honors "struct:field:name" meta over the given name.
   In-tree coverage: deleted tests only. Inferable: partially —
   documented on the declaration.
5. WrapText wraps at maxChars keeping words intact, preferring a break
   at the last space inside the window and falling back to the next
   space; existing newlines are preserved. In-tree coverage: deleted
   tests only. Inferable: partially.
6. Indent prefixes every non-empty line with the prefix. In-tree
   coverage: none directly. Inferable: yes.
7. Comment joins all element lines, trims leading whitespace per line,
   wraps at 77 columns, and indents with "// ". In-tree coverage:
   deleted tests only. Inferable: partially — the 77-column width is
   arbitrary.
8. Goify results are cached per (input, firstUpper, acronym) and per
   operation in a process-global cache. In-tree coverage: none.
   Inferable: no — caching is an implementation contract.
