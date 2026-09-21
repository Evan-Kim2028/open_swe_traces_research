"""The gate: one rule registry with evidence tiers.

executed > static > documentary — a lower tier never overwrites a higher-tier
verdict for the same rule; it is retained in ``rule_verdicts_secondary``.

Importing this package registers every rule impl:

- ``legacy``: synth.rules checks (documentary/static honest tiers)
- ``exec_rules``: in-image bare/gold/cheat suites (executed)
- ``static_rules``: coverage + level nesting (static)
- ``contract_gold``: A13 contract-vs-gold consistency (static)
- ``alt_fix``: passed-trial patch vs gold (executed A2)
"""

from openswe_traces.gate import (  # noqa: F401 — registers impls
    alt_fix,
    contract_gold,
    exec_rules,
    legacy,
    static_rules,
)
from openswe_traces.gate.context import GateContext, level_dirs
from openswe_traces.gate.core import (
    DOCUMENTARY,
    EXECUTED,
    STATIC,
    TIER_ORDER,
    GateError,
    Verdict,
    executed_rule_ids,
    load_validation,
    merge_verdict_rows,
    record_verdicts,
    register,
    registry,
    rule_ids,
    stored_verdict_rows,
    verdict_from_dict,
    write_validation,
)
from openswe_traces.gate.runner import (
    GateReport,
    attach_payload,
    ensure_gate,
    evaluate_gate,
    gate_blockers,
    gate_task,
    write_task_validation,
)

__all__ = [
    "DOCUMENTARY",
    "EXECUTED",
    "STATIC",
    "TIER_ORDER",
    "GateContext",
    "GateError",
    "GateReport",
    "Verdict",
    "attach_payload",
    "ensure_gate",
    "evaluate_gate",
    "executed_rule_ids",
    "gate_blockers",
    "gate_task",
    "level_dirs",
    "load_validation",
    "merge_verdict_rows",
    "record_verdicts",
    "register",
    "registry",
    "rule_ids",
    "stored_verdict_rows",
    "verdict_from_dict",
    "write_task_validation",
    "write_validation",
]
