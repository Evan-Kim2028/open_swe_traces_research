1. An empty key writes nothing (no bytes, no CRLF). Inferable: yes
2. When `n > 0` the line starts with `LS+ ` followed by the key. Inferable: yes
3. When `n > 0` and the key contains a space (queue group), a decimal encoding of `n` is appended after another space. Inferable: doc
4. The decimal encoding of `n` is written with a small stack buffer walking `l /= 10` and `digits[l%10]`, most significant digit first, with no leading zeros. Inferable: yes
5. Queue count is omitted when the key has no space, even if `n > 1`. Inferable: doc
6. When `n <= 0` the line is `LS- ` plus the key, with no count. Inferable: yes
7. Every successful write (non-empty key) ends with `CR_LF`. Inferable: yes
8. When tracing is on, `LS+` traces `"key n"` for queue keys and just `key` otherwise; `LS-` traces the key. Inferable: yes
