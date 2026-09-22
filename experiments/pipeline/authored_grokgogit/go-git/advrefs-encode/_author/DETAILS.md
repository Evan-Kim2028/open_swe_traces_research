1. Protocol V1 emits a leading `version 1\n` pkt-line. V0 emits none. Any other version is an error. Inferable: partially
2. The first ref line is HEAD if a non-peeled HEAD exists, otherwise the first non-peeled ref in `References` order. Inferable: no
3. With no non-peeled refs, the first line uses the zero hash and the name `capabilities^{}`. Inferable: no
4. The first line is `<hash> SP <name> NUL <caps> LF` even when caps are empty (NUL still present). Inferable: no
5. Remaining non-peeled refs, excluding the one used as the first line, are sorted by name and written as `<hash> SP <name> LF`. Inferable: doc
6. A peeled ref `name^{}` is written immediately after `name`, not in the sorted name order of the peeled name. Inferable: doc
7. Shallows are written after refs as `shallow <hash>\n`, sorted by hex string. Inferable: yes
8. The advertisement ends with a flush packet. Inferable: yes
9. Every data payload ends with a newline. Inferable: doc
