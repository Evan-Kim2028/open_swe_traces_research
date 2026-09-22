1. Zero wants returns an error. Inferable: yes
2. Wants are sorted with `plumbing.HashesSort` before writing. Inferable: partially
3. Consecutive duplicate wants (after sort) are skipped. Inferable: partially
4. The first want line is `want <hash>\n`, or `want <hash> <caps>\n` when capabilities are non-empty. Inferable: yes
5. Later wants are `want <hash>\n` with no capabilities. Inferable: yes
6. Shallows are sorted, consecutive duplicates skipped, and written as `shallow <hash>\n` after the wants. Inferable: partially
7. `Deepen > 0` together with a non-zero `DeepenSince` or a non-empty `DeepenNot` returns `ErrDeepenMutuallyExclusive`. Inferable: doc
8. `deepen <n>` is omitted when `Deepen` is 0. Inferable: yes
9. `deepen-since` uses `DeepenSince.UTC().Unix()`. Inferable: no
10. Each `DeepenNot` ref is a `deepen-not <ref>\n` line. Inferable: yes
11. A non-empty `Filter` is `filter <filter>\n`. Inferable: yes
12. The message ends with a flush packet. Inferable: yes
13. Every data payload ends with a newline. Inferable: doc
