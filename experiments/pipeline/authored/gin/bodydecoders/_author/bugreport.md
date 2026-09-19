# Bug report

binding a request body fails for every content type: decoders panic or return errors on valid payloads, so JSON/XML/YAML/protobuf posts can never populate the target struct.
Reproduce with:

```
tests/test.sh
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository and test output.
