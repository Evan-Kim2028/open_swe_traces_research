1. An `int64` input is returned unchanged with a nil error. Inferable: yes
2. Any other non-string type is an error. Inferable: yes
3. An empty string returns `(0, nil)`. Inferable: no
4. A non-empty string is split as `prefix + lastChar`; `prefix` is parsed with `strconv.ParseInt(..., 10, 64)`. Inferable: yes
5. A prefix that is not a base-10 int64 returns that parse error. Inferable: yes
6. Suffix `K` multiplies by `1<<10`. Inferable: partially
7. Suffix `M` multiplies by `1<<20`. Inferable: partially
8. Suffix `G` multiplies by `1<<30`. Inferable: partially
9. Suffix `T` multiplies by `1<<40`. Inferable: partially
10. Any other last character is an error. Inferable: yes
11. Suffix letters are case-sensitive; lowercase is not accepted. Inferable: no
12. There is no space between the number and the suffix. Inferable: yes
