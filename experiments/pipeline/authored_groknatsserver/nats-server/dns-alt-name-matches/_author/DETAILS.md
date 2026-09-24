1. A nil URL in the slice is skipped. Inferable: yes
2. The URL hostname is split on `.` after `strings.ToLower`. Inferable: yes
3. The SAN and the hostname must have the same number of labels; otherwise that URL cannot match. Inferable: doc
4. A SAN whose first label is exactly `"*"` may skip comparing label 0 and compare the rest. Inferable: doc
5. A `"*"` in any later SAN label is compared as a literal, not a wildcard. Inferable: doc
6. `"*"` never spans more than one label, so `*.a.b` cannot match `x.y.a.b`. Inferable: doc
7. Remaining labels are compared with `!=` after both sides are already lowercase. Inferable: yes
8. The first URL that survives the label walk returns true. Inferable: yes
9. If no URL matches, the result is false. Inferable: yes
10. An empty hostname (no labels other than a possible empty string from `"."`) still participates in the equal-length check. Inferable: partially
