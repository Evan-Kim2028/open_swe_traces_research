# Bug report

Role names no longer parse correctly: valid roles are rejected, while misspelled or pluralised spellings are silently accepted in contexts that should reject them. In lenient mode the legacy master spelling does not resolve. Separately, raw YAML parsing accepts documents containing unknown fields instead of rejecting them, and empty input produces an error instead of succeeding as a no-op.

Expected: `control-plane`, `node`, `bastion`, `apiserver` resolve strictly; `nodes` and `controlplane` resolve only in lenient mode; `master` resolves only in lenient mode; unknown fields in a YAML document are an error.

Reproduce with:

```
go test -count=1 ./pkg/apis/kops/
```

Work in `/app`. Keep unrelated tests passing. Do not use web search or any tool that accesses the internet; work only from the repository.
