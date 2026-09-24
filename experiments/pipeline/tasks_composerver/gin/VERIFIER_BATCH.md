# VERIFIER_BATCH.md — composerver gin

Composer verifier batch for gin/httprouter feature-excision units.
Seed `20260919`. Hidden black-box property suites via `affordance.py`.
Dockerfile `FROM ladder-base:gin`. L0/L2 proved; L5/L6 packaged only.

Build / reprove:

```
uv run python scripts/build_composerver_gin_batch.py --max-parallel 2
```

## Summary

| unit | properties | coverage % | gold 1st | suite fixes | wall min | verdict |
|---|---:|---:|---|---:|---:|---|
| `bindingdispatch` | 9 | 100.0 | yes | 0 | 0.4 | PASS |

**Batch totals:** 1/1 PASS

---

## `bindingdispatch`

- **Properties:** 9
- **Contract coverage:** 100.0%
- **Gold first proof:** yes
- **Suite fixes:** 0
- **Wall minutes:** 0.4

- **L0:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L0`
- **L2:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L2`
- **L5:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L5` (packaged, not proved)
- **L6:** `experiments/pipeline/tasks_composerver/gin/bindingdispatch-L6` (packaged, not proved)

### Coverage (contract sentence → property)

| contract sentence | property |
|---|---|
| every MIME row of the dispatch table returns its binder; GET overrides content type | `TestBBDefaultGetAlwaysForm` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultGetOverridesMIME` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultMIMETable` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultDualMIMEAliases` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultFormFallback` |
| with no validator installed the validate hook is a no-op | `TestBBDefaultUnknownMIME` |
| with no validator installed the validate hook is a no-op | `TestBBValidationDisabledNilValidator` |
| with no validator installed the validate hook is a no-op | `TestBBValidationDisabledRandomBodies` |
| with no validator installed the validate hook is a no-op | `TestBBValidationEnabledRequiresField` |

### Proofs

| check | result |
|---|---|
| `proof_harness` | pass |
| `buggy_fails` | pass |
| `gold_restore` | pass |
| `cheat_rejected` | pass |
| `blackbox_hygiene` | pass |

