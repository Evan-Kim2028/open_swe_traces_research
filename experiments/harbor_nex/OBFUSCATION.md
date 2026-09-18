# harbor_nex obfuscation

Date: 2026-09-18. Identity obfuscation of six client-go Harbor tasks so
server-side web search/fetch of `tikv/client-go` cannot recall the fix.
Module path `github.com/tikv/client-go/v2` → `example.internal/kvstore/v2`.
Brand text stripped; subsystem symbols renamed via `gopls rename`.
Did not launch a Harbor job.

Reproduce:

```
uv run python scripts/obfuscate_task.py --batch
```

Output: `experiments/harbor_nex/tasks_obf/<name>-obf/`.

## client-go-keyspacecodec-obf

- mapping size: **90**
- notes: one-function alt cannot pass f2p at this size

| check | result |
|---|---|
| docker | skipped |
| tests adjusted (semantic) | none |

Searchability (new identifiers vs upstream checkout):

| ident | upstream matches | ok |
|---|---:|---|
| `NimbusClip` | 0 | True |
| `NimbusCore` | 0 | True |
| `NimbusGate` | 0 | True |
| `NimbusJoin` | 0 | True |
| `NimbusPack` | 0 | True |

No test semantics changed (checksums refreshed after rename/import rewrite).

## client-go-memget-obf

- mapping size: **49**
- notes: naive linear alt fails the perf gate

| check | result |
|---|---|
| docker | skipped |
| tests adjusted (semantic) | none |

Searchability (new identifiers vs upstream checkout):

| ident | upstream matches | ok |
|---|---:|---|
| `NimbusSeal` | 0 | True |
| `PebbleNode` | 0 | True |
| `PebblePipe` | 0 | True |
| `PebbleRing` | 0 | True |
| `PebbleSeal` | 0 | True |

No test semantics changed (checksums refreshed after rename/import rewrite).

## client-go-batchcmds-obf

- mapping size: **163**

| check | result |
|---|---|
| docker | skipped |
| tests adjusted (semantic) | none |

Searchability (new identifiers vs upstream checkout):

| ident | upstream matches | ok |
|---|---:|---|
| `NimbusCore` | 0 | True |
| `NimbusGate` | 0 | True |
| `NimbusPort` | 0 | True |
| `NimbusSlot` | 0 | True |
| `PebbleCore` | 0 | True |

No test semantics changed (checksums refreshed after rename/import rewrite).

## client-go-interceptor-obf

- mapping size: **26**

| check | result |
|---|---|
| docker | skipped |
| tests adjusted (semantic) | none |

Searchability (new identifiers vs upstream checkout):

| ident | upstream matches | ok |
|---|---:|---|
| `NimbusRing` | 0 | True |
| `WillowBolt` | 0 | True |
| `WillowRing` | 0 | True |
| `YarrowPort` | 0 | True |
| `AmberSeal` | 0 | True |

No test semantics changed (checksums refreshed after rename/import rewrite).

## client-go-onepc-scope-obf

- mapping size: **69**

| check | result |
|---|---|
| docker | skipped |
| tests adjusted (semantic) | none |

Searchability (new identifiers vs upstream checkout):

| ident | upstream matches | ok |
|---|---:|---|
| `NimbusBolt` | 0 | True |
| `NimbusClip` | 0 | True |
| `NimbusFold` | 0 | True |
| `NimbusNode` | 0 | True |
| `PebbleBolt` | 0 | True |

No test semantics changed (checksums refreshed after rename/import rewrite).

## client-go-memsetvalue-obf

- mapping size: **37**

| check | result |
|---|---|
| docker | skipped |
| tests adjusted (semantic) | none |

Searchability (new identifiers vs upstream checkout):

| ident | upstream matches | ok |
|---|---:|---|
| `PebbleNode` | 0 | True |
| `PebblePipe` | 0 | True |
| `PebbleRing` | 0 | True |
| `PebbleSeal` | 0 | True |
| `PebbleUnit` | 0 | True |

No test semantics changed (checksums refreshed after rename/import rewrite).
