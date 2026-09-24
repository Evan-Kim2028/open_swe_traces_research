# Difficulty — tablesfmt

predicted_flip: L4
details: 7
Commitments: reflection-based getter invocation, name-keyed column resolution with first-missing error, closure-to-sort.Interface adaptation, lexicographic multi-column row sort, fatal-on-non-slice, and tabwriter escaping/alignment parameters.

Hardness driver: the output is whitespace-exact (escaped cells, tab stops, alignment), the sort is a multi-key string comparison hidden inside a swap/less adapter, and non-slice input fatals rather than errors.
