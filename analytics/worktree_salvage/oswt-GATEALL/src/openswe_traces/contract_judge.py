"""LLM judge for A13 — contract-vs-gold consistency.

One request per unit over the OpenRouter free tier. Each request carries the
coverage rows, the contract prose, the gold patch, and the hidden suite's
tests (names, fatal messages, bodies truncated). The response is a JSON
verdict::

    {
      "rows":    [{"test": "TestX", "verdict": "ok|false|vague",
                   "reason": "..."}],
      "missing": [{"test": "TestHidden", "assertion": "..."}]
    }

``rows.verdict == "false"`` means the row's claim is untrue of ``gold.patch``.
``missing`` lists hidden-suite assertions no coverage row or contract
sentence states. Verdicts cache to ``outputs/contract_judge/<key>.json``
keyed on the evidence hash, so contract edits invalidate automatically and
re-runs resume where they stopped. The gate (``gate.contract_gold``) only
ever *reads* this cache; it never touches the network itself.

The audit dump (``outputs/contract_gold_dump/``) is deliberately not read
here: the judge's value is independent reproduction of the 20 hand-found
defects, so nothing in the prompt may come from the answer key.
"""

from __future__ import annotations

import json
import re
import time
import urllib.error
import urllib.request
from collections.abc import Iterable
from dataclasses import dataclass
from pathlib import Path

from openswe_traces.data import ROOT
from openswe_traces.gate.contract_gold import (
    PROMPT_VERSION,
    UnitEvidence,
    judge_cache_path,
)

OPENROUTER_URL = "https://openrouter.ai/api/v1/chat/completions"
# Free-tier models reachable on this key (probed 2026-09), tried in order
# on 404/"no endpoints" errors. nex-n2.5-pro is the repo-proven one.
MODELS = (
    "nvidia/nemotron-3-super-120b-a12b:free",
    "nvidia/nemotron-3-ultra-550b-a55b:free",
    "nex-agi/nex-n2.5-pro:free",
    "google/gemma-4-31b-it:free",
)
MAX_REQUESTS = 300
MAX_GOLD_CHARS = 28_000
MAX_BODY_CHARS = 1_600
MAX_TOTAL_CHARS = 90_000
REQUEST_TIMEOUT = 240
MAX_ATTEMPTS = 4

_RESPONSE_RE = re.compile(r"\{.*\}", re.DOTALL)


@dataclass
class JudgeOutcome:
    key: str
    verdict: dict | None  # parsed judgement, or None on failure
    model: str
    requests: int  # requests consumed by this unit
    error: str | None = None


def load_api_key(env_path: Path | None = None) -> str:
    path = env_path or ROOT / ".env"
    if not path.is_file():
        raise SystemExit(f"A13 judge: no {path} — needs OPENROUTER_API_KEY")
    for line in path.read_text().splitlines():
        line = line.strip()
        if line.startswith("OPENROUTER_API_KEY="):
            key = line.split("=", 1)[1].strip().strip('"').strip("'")
            if key:
                return key
    raise SystemExit(f"A13 judge: OPENROUTER_API_KEY not found in {path}")


def _truncate(text: str, cap: int) -> str:
    if len(text) <= cap:
        return text
    head = cap * 3 // 4
    return text[:head] + f"\n...[{len(text) - cap} chars elided]...\n" + text[-(cap - head) :]


def build_prompt(ev: UnitEvidence) -> str:
    rows_block = "\n".join(
        f"  ROW {i}: original test `{r.test}` claims: {r.sentence}"
        for i, r in enumerate(ev.rows, 1)
    ) or "  (no coverage rows)"
    map_block = "\n".join(
        f'  "{e.phrase}" -> {", ".join(e.tests)}' for e in ev.map_entries
    ) or "  (no contract->property map in the suite header)"
    tests_block = "\n\n".join(
        f"### {t.name}\nassertion messages: "
        + ("; ".join(t.fatals) if t.fatals else "(none)")
        + ("\n```go\n" + _truncate(t.body, MAX_BODY_CHARS) + "\n```" if t.body else "")
        for t in ev.hidden
    )
    return f"""You are auditing a synthetic SWE task. The task ships a CONTRACT
(prose plus a coverage table) describing behavior the reference fix
(gold.patch) implements, and a HIDDEN test suite that grades submissions.

Decide two things:

1. For each coverage ROW below: is its claim TRUE of the code changes in
   gold.patch? Verdicts:
   - "ok":    the claim is true of gold (or trivially consistent).
   - "false": the claim contradicts gold's actual behavior — e.g. wrong
     error semantics, wrong format, wrong boundary, claims gold does
     something it does not.
   - "vague": the row names a topic but states no checkable claim, so its
     truth cannot be determined.

2. MISSING coverage: hidden-test assertions that NO coverage row and NO
   contract sentence states. Only list an assertion when the hidden test
   *fails* unless the implementation has that specific behavior — do not
   list behavior that follows from a stated row, generic API usage, or
   fuzz/property loops exercising already-covered behavior.

CONTRACT PROSE:
{ev.prose}

COVERAGE ROWS (first column is the ORIGINAL upstream test name; the claim
is what matters, not the name):
{rows_block}

SUITE CONTRACT->PROPERTY MAP (author's own row-to-test annotation):
{map_block}

GOLD PATCH:
```diff
{_truncate(ev.gold, MAX_GOLD_CHARS)}
```

HIDDEN TESTS:
{tests_block}

Answer with JSON only, no prose outside the object:
{{"rows": [{{"test": "<row test>", "verdict": "ok|false|vague",
  "reason": "<one sentence citing gold behavior>"}}],
  "missing": [{{"test": "<hidden test>", "assertion": "<unstated behavior>"}}]}}
Include every row exactly once. "missing" may be empty."""


def attach_bodies(ev: UnitEvidence, hidden_files: dict[str, str]) -> None:
    """Fill HiddenTest.body from raw files (kept out of UnitEvidence itself)."""
    bodies: dict[str, str] = {}
    for text in hidden_files.values():
        funcs = list(re.finditer(r"^func\s+(Test[A-Za-z0-9_]+)\s*\(", text, re.MULTILINE))
        for i, m in enumerate(funcs):
            end = funcs[i + 1].start() if i + 1 < len(funcs) else len(text)
            bodies[m.group(1)] = text[m.start() : end]
    for t in ev.hidden:
        t.body = bodies.get(t.name, "")


def _extract_json(text: str) -> dict | None:
    m = _RESPONSE_RE.search(text)
    if not m:
        return None
    try:
        data = json.loads(m.group(0))
    except json.JSONDecodeError:
        return None
    return data if isinstance(data, dict) else None


def _post(api_key: str, model: str, prompt: str) -> tuple[dict | None, str | None]:
    payload = json.dumps(
        {
            "model": model,
            "messages": [{"role": "user", "content": prompt}],
            "temperature": 0,
            "max_tokens": 16384,
        }
    ).encode()
    req = urllib.request.Request(
        OPENROUTER_URL,
        data=payload,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://devin.ai",
            "X-Title": "oswt contract-gold A13 judge",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=REQUEST_TIMEOUT) as resp:
            data = json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        body = e.read().decode(errors="replace")[:300]
        return None, f"HTTP {e.code}: {body}"
    except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as e:
        return None, f"{type(e).__name__}: {e}"
    try:
        content = data["choices"][0]["message"]["content"]
    except (KeyError, IndexError, TypeError):
        return None, f"malformed response: {str(data)[:300]}"
    verdict = _extract_json(content or "")
    if verdict is None:
        return None, f"unparseable content: {str(content)[:300]}"
    return verdict, None


def judge_unit(
    ev: UnitEvidence,
    api_key: str,
    models: Iterable[str] = MODELS,
    budget: int = MAX_REQUESTS,
    cache_dir: Path | None = None,
) -> JudgeOutcome:
    """Judge one unit; write the verdict to the cache. Resumes via key."""
    path = judge_cache_path(ev.contract_key, cache_dir)
    if path.is_file():
        try:
            data = json.loads(path.read_text())
            return JudgeOutcome(ev.contract_key, data.get("verdict", data),
                                str(data.get("model", "cached")), 0)
        except (OSError, json.JSONDecodeError):
            pass  # corrupt cache entry — re-judge

    prompt = _truncate(build_prompt(ev), MAX_TOTAL_CHARS)
    requests = 0
    last_err = "no models configured"
    models = list(models)
    for attempt in range(MAX_ATTEMPTS):
        model = models[min(attempt, len(models) - 1)]
        if requests >= budget:
            return JudgeOutcome(ev.contract_key, None, model, requests,
                                "request budget exhausted")
        requests += 1
        verdict, err = _post(api_key, model, prompt)
        if verdict is not None:
            record = {
                "key": ev.contract_key,
                "prompt_version": PROMPT_VERSION,
                "model": model,
                "unit": f"{ev.repo}/{ev.unit}",
                "verdict": verdict,
            }
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(json.dumps(record, indent=1) + "\n")
            return JudgeOutcome(ev.contract_key, verdict, model, requests)
        last_err = err
        # 404 / "no endpoints" → next model; rate limit → back off, same model
        if err and ("404" in err or "No endpoints" in err):
            continue
        if err and "429" in err:
            time.sleep(20 * (attempt + 1))
            continue
        time.sleep(5)
    return JudgeOutcome(ev.contract_key, None, models[-1], requests, last_err)


def judge_units(
    units: Iterable[tuple[Path, UnitEvidence]],
    api_key: str,
    models: Iterable[str] = MODELS,
    max_requests: int = MAX_REQUESTS,
    cache_dir: Path | None = None,
    log=None,
) -> list[JudgeOutcome]:
    """Judge a bank of ``(task_dir, evidence)`` pairs. Resume-safe: cached
    keys skip the network. A unit that fails after retries is logged and
    skipped; the loop stops only when the request budget is spent."""
    out: list[JudgeOutcome] = []
    spent = 0
    for task_dir, ev in units:
        attach_bodies(ev, load_hidden_files(task_dir))
        outcome = judge_unit(ev, api_key, models, max_requests - spent,
                             cache_dir)
        spent += outcome.requests
        out.append(outcome)
        if log:
            status = "cached" if outcome.requests == 0 else (
                "ok" if outcome.verdict is not None
                else f"FAILED {outcome.error}"
            )
            log(f"{ev.repo}/{ev.unit}: {status} ({outcome.model}, "
                f"spent={spent}/{max_requests})")
        if spent >= max_requests:
            break
    return out


def load_hidden_files(task_dir: Path) -> dict[str, str]:
    root = task_dir / "tests" / "hidden"
    out: dict[str, str] = {}
    if root.is_dir():
        for f in sorted(root.rglob("*.go")):
            out[str(f.relative_to(root))] = f.read_text(errors="replace")
    return out
