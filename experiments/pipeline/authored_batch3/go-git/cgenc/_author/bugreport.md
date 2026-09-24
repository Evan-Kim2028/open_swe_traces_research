# Bug report

Commit-graph files written by the encoder are unreadable. The chunk
offset table points at the wrong places — offsets are relative instead of
absolute or forget the header size. The fanout table counts hashes per
bucket instead of cumulatively, and hashes are written in caller order
rather than sorted. Commits with three or more parents lose the extra
parents entirely or write the marker bits into the wrong slot, generation
values above the 31-bit inline limit corrupt neighbouring rows instead of
spilling to the overflow chunk, and the generation field lands in the
wrong bits of the packed timestamp word. The trailing checksum is missing
or covers only part of the file.

Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any
tool that accesses the internet; work only from the repository and test
output.

IMPORTANT: This repository is fully self-contained. Do NOT use web search, web fetch, or any tool that accesses the internet, and do not attempt to download or consult upstream sources; any such use disqualifies the attempt. Work only from the files in the repository and the test output.
