# Difficulty — renamedet

predicted_flip: L3
details: 14

Missed edges: both-sides precondition; hash+mode exact gate; best-name
tiebreak; bounded greedy matrix matching; 25/25/50 name weights; only-
exact and rename-limit refusal; regular-file-only filter; size+1 dodge;
99:1 content:name blend; text-vs-binary region hashing and CRLF fold;
index growth to the 30-bit cap; symmetry + empty-file max score; final
stable-sort ordering.

Hardness driver: a scoring pipeline with three distinct weighting
constants — wrong constants still produce *plausible* rename sets, so
guesses pass weak tests and fail the table cases.
