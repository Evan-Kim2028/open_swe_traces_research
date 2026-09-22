1. A section that has options is written as `[name]` on its own line, then its options, then its subsections. Inferable: yes
2. A section with no options and only subsections omits the bare `[name]` header. Inferable: partially
3. A subsection header is `[section "name"]` with the subsection name escaped by a replacer that turns `"` into `\"` and `\` into `\\`. Inferable: no
4. Each option is written as a tab, the key, ` = `, the value, and a newline. Inferable: yes
5. The value is wrapped in double quotes when it contains any of `# ; " tab newline backslash`, or when it has a leading or trailing space. Inferable: no
6. Inside a quoted value, `"` becomes `\"`, `\` becomes `\\`, newline becomes `\n`, tab becomes `\t`, backspace becomes `\b`. Inferable: no
7. Values with other unusual bytes (control characters, non-ASCII) are written unquoted if they miss the trigger set. Inferable: no
8. Duplicate keys are emitted in order, each on its own line. Inferable: yes
