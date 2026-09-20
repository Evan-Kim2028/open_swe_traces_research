# Difficulty — negside

predicted_flip: L2
details: 10

Missed edges: shallow stream tolerating missing flush while push options require it; hashes
taken from fixed-length line tails with no hex validation; options written with no newline;
validate-all-before-write; space being a graphic character; nil → empty slice on decode.

Hardness driver: two codecs whose flush/validation choices deliberately differ; several details
contradict the "obvious" implementation.
