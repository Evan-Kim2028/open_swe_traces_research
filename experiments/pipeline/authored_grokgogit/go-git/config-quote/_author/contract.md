# Contract — config-quote

Hidden suite: `tests/hidden/plumbing/format/config/config_quote_bb_test.go`
(package `config`, in-package). One `TestDetailNN` per DETAILS.md line.

| test | detail | inferable | asserts |
|---|---|---|---|
| TestDetail01 | 1 | yes | A section with options emits `[name]` on its own line, then its option lines, then its subsection blocks, in that order. |
| TestDetail02 | 2 | partially | A section with no options and only subsections emits no bare `[name]` header; the subsection header still appears. |
| TestDetail03 | 3 | no | SHAPE: subsection headers render as `[section "name"]`; `"` in the name appears backslash-escaped and `\` appears doubled. |
| TestDetail04 | 4 | yes | Each option line is a tab, the key, ` = `, the value, and a newline. |
| TestDetail05 | 5 | no | SHAPE: values containing `#`, `;`, `"`, tab, newline, backslash, or a leading/trailing space are emitted inside double quotes; a plain value and an apostrophe-containing value are not. |
| TestDetail06 | 6 | no | SHAPE: inside a quoted value, `"` renders as `\"`, `\` as `\\`, newline as `\n`, tab as `\t`, and backspace as `\b`. |
| TestDetail07 | 7 | no | SHAPE: values whose unusual bytes miss the trigger set (a control byte, a non-ASCII string, DEL) are emitted verbatim and unquoted. |
| TestDetail08 | 8 | yes | Three options sharing a key emit three lines in input order. |

Refusals/softening: line 3 asserts the escaping only for the two committed
characters; other bytes in subsection names are left free. Line 5 exercises
every committed trigger plus two committed non-triggers; the treatment of
other boundary whitespace is left free. Line 6 pins the five committed
escapes only. Line 7 asserts presence-verbatim of off-set bytes; it does not
pin how an implementation orders or escapes beyond the committed set.
