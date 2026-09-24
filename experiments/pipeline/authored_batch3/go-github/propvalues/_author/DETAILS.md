# DETAILS — propvalues

1. `DefaultValueString` succeeds for string, single_select and url value
   types when the payload actually holds a string; any other type returns
   false. Inferable: partially — the accessor set is fixed, which value
   types map to it is an API convention.
2. `DefaultValueStrings` succeeds only for multi_select, accepting both a
   real []string and a []any whose items are all strings. Inferable:
   partially — the []any tolerance is a decoding compromise, arbitrary.
3. `DefaultValueBool` succeeds only for true_false when the payload is a
   string parseable as a boolean. Inferable: partially — accepting only
   string payloads (not bool) is arbitrary.
4. Every accessor reports (value, false) rather than guessing when the
   value type or payload shape does not match. Inferable: yes — the
   comma-ok signatures force it.
