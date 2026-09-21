"""Adapter: register every synth.rules check once, at its honest tier.

``documentary``: the verdict is copied from notes / recorded rewards /
hand-maintained fields (``ctx.md_checks``, ``parse_harbor_notes`` …).
``static``: the verdict is derived from packaged artifacts (task.toml,
test.sh, instruction text, hidden tests).

The gate's executed impls (exec_rules) and packaging impls (static_rules)
register separately; a rule may have impls at several tiers — the merge keeps
the strongest.
"""

from __future__ import annotations

from openswe_traces.gate.context import GateContext
from openswe_traces.gate.core import DOCUMENTARY, STATIC, Verdict, _now_iso, register
from openswe_traces.synth.rules import RULE_IDS, RULES, RuleVerdict

# Static: the check inspects packaged files, not notes. B7 lives in
# gate.static_rules (excision-aware); the legacy impl stays documentary.
STATIC_RULES = frozenset(
    {"A10", "A12", "B1", "B2", "B3", "B4", "B5", "B6", "B8"}
)


def _adapt(rule_id: str):
    check = RULES[rule_id]
    tier = STATIC if rule_id in STATIC_RULES else DOCUMENTARY

    def run(ctx: GateContext) -> Verdict:
        rv: RuleVerdict = check(ctx.rules)
        return Verdict(
            rule_id=rule_id,
            passed=rv.passed,
            skipped=rv.skipped,
            tier=tier,
            evidence=rv.evidence,
            produced_at=_now_iso(),
            provenance=f"synth.rules:{check.__name__}",
        )

    return run


for _rid in RULE_IDS:
    register(
        _rid,
        STATIC if _rid in STATIC_RULES else DOCUMENTARY,
        provenance="synth.rules",
        description=RULES[_rid].__doc__ or "",
    )(_adapt(_rid))
