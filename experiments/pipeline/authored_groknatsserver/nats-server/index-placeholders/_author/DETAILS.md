1. A token of length 0 or 1 is `NoTransform`, indexes `[]int{-1}`, scalar `-1`, empty string, nil error. Inferable: partially
2. A token starting with `$` whose remainder parses as base-10 int is `Wildcard` with that single index, scalar `-1`, empty string, nil error. Inferable: yes
3. A token starting with `$` whose remainder is not an int is `NoTransform` with indexes `[]int{-1}`, scalar `-1`, empty string, nil error (not a parse error). Inferable: doc
4. Mustache form is recognized only when length > 4, the token starts with `{{`, and it ends with `}}`. Inferable: yes
5. `wildcard` with a single empty argument is `BadTransform` and a not-enough-args mapping error wrapping the original token. Inferable: yes
6. `wildcard` with exactly one integer argument is `Wildcard` with that index, scalar `-1`. Inferable: yes
7. `wildcard` with more than one argument is `BadTransform` and a too-many-args mapping error. Inferable: yes
8. `partition` with a single integer argument that fits in int32 is `Partition`, empty index slice, that integer as the scalar. Inferable: doc
9. `partition` with first argument N and following integer token indexes is `Partition` with those indexes in order and scalar N. Inferable: doc
10. A `partition` or `random` integer larger than `math.MaxInt32` is `BadTransform` and an invalid-arg mapping error. Inferable: yes
11. `SplitFromLeft`, `SplitFromRight`, `SliceFromLeft`, `SliceFromRight`, `Left`, and `Right` are dispatched through `transformIndexIntArgsHelper` with the matching kind constant. Inferable: yes
12. `split` requires exactly two arguments; the first is an integer token index and the second is the delimiter. Inferable: yes
13. A `split` delimiter that contains a space or the subject token separator is `BadTransform` and an invalid-arg mapping error. Inferable: no
14. `random` requires exactly one integer argument; any other arity is `BadTransform` and a not-enough-args mapping error. Inferable: partially
15. A well-formed `{{...}}` token that matches no known mapping function is `BadTransform` and an unknown-function mapping error wrapping the original token. Inferable: yes
16. Mapping function names inside mustache tokens are matched case-insensitively and allow internal whitespace, because the remaining regexes already encode that. Inferable: yes
17. Integer arguments are parsed after trimming spaces. Inferable: yes
18. On any successful non-split mapping the string argument is empty. Inferable: yes
