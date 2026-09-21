#!/usr/bin/env python3
"""Check a candidate excision closure against closures the bank already owns.

Two data sources:

1. ``experiments/pipeline/authored*/**/_author/closure.md`` — the authored bank.
   Parses the ``Removed functions`` bullet and any ``File``/``Package`` lines.
2. The legacy client-go task set (``experiments/harbor_nex/tasks_*``) — embedded
   below as a static table mapping file -> excised symbols, derived by hashing
   every ``.go`` file across the task env trees and diffing the outliers.

Usage:
  check_unit_overlap.py --spec excise_spec.json
  check_unit_overlap.py FILE FUNC [FUNC ...]
  check_unit_overlap.py --funcs FILE            # list every func in FILE, TAKEN/FREE
  check_unit_overlap.py --funcs FILE --tree /path/to/src
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

# file -> {symbol: task-name}.  Derived 2026-09-21 by hashing all .go files
# across experiments/harbor_nex/tasks_all/client-go-*/environment/src and
# locating the enclosing function of every changed line vs the modal version.
# Paths are upstream spellings; the obfuscated tree renames tikvrpc->wirerpc,
# internal/mockstore/mocktikv->internal/mockstore/mockkv, internal/retry->
# internal/client/retry.
TASK_EXCISIONS = {
    "config/config.go": {"GetGlobalConfig": "getglobalconfig"},
    "config/retry/backoff.go": {"NewBackofferWithVars": "newbackofferwithvars"},
    "config/retry/config.go": {"expo": "dualexpo"},
    "internal/client/retry/config.go": {"expo": "dualexpo"},
    "error/error.go": {"IsErrNotFound": "iserrnotfound"},
    "internal/apicodec/codec.go": {
        "DecodeKey": "decodekeyv1+keyspacecodec",
        "ParseKeyspaceID": "keyspacecodec+keyspaceidcodec",
        "attachAPICtx": "keyspacecodec",
    },
    "internal/apicodec/codec_v1.go": {"codecV1.decodeRegionError": "keyspacecodec"},
    "internal/apicodec/codec_v2.go": {
        "*": "keyspacecodec+decodebucketkeys+keyspaceidcodec+keyspaceprefix (whole file)",
    },
    "internal/apicodec/mem_codec.go": {"IsDecodeError": "keyspacecodec"},
    "internal/client/client.go": {"RPCClient.sendRequest": "batchcmds"},
    "internal/client/client_batch.go": {
        "batchCommandsBuilder.buildWithLimit": "batchcmds",
        "batchCommandsBuilder.hasHighPriorityTask": "batchcmds",
        "batchCommandsBuilder.len": "batchcmds",
        "batchConn.fetchAllPendingRequests": "batchcmds",
        "batchConn.fetchMorePendingRequests": "batchcmds",
        "sendBatchRequest": "batchcmds",
    },
    "internal/client/client_interceptor.go": {
        "buildResourceControlInterceptor": "interceptor",
        "interceptedClient.SendRequest": "interceptor",
    },
    "internal/locate/pd_codec.go": {
        "GetKeyspaceID": "keyspacecodec",
        "NewCodecPDClientWithKeyspace": "keyspacecodec",
    },
    "internal/locate/region_cache.go": {"KeyLocation.Contains/interval": "intervalcontains"},
    "internal/locate/region_request.go": {
        "IsFakeRegionError": "isfakeregionerror",
        "NewRegionRequestSender": "newregionrequestsender",
    },
    "internal/mockstore/mockkv/rpc.go": {"RPCClient.SendRequest": "batchcmds"},
    "internal/unionstore/memdb.go": {
        "MemDB.Get": "memget",
        "MemDB.set": "memget",
        "MemDB.setValue": "memsetvalue",
    },
    "oracle/oracle.go": {
        "ExtractPhysical": "extractphysical",
        "GetPhysical": "getphysical",
        "GetTimeFromTS": "gettimefromts",
    },
    "oracle/oracles/local.go": {"localOracle.GetTimestamp": "gettimestamp"},
    "wirerpc/endpoint.go": {"GetStoreTypeByMeta": "getstoretypebymeta"},
    "wirerpc/interceptor/interceptor.go": {
        "RPCInterceptorChain.Link": "interceptor",
        "RPCInterceptorChain.Wrap": "interceptor",
    },
    "wirerpc/tikvrpc.go": {
        "Request.ToBatchCommandsRequest": "batchcmds",
        "NewRequest": "newrequest",
    },
    "txnkv/transaction/2pc.go": {"twoPhaseCommitter.checkOnePC": "onepc-scope"},
}

REMOVED_RE = re.compile(r"Removed functions.*?:\s*(.+)", re.I)
FUNC_RE = re.compile(r"^func\s+(?:\(([^)]*)\)\s*)?([A-Za-z_]\w*)")


def norm_path(p: str) -> str:
    """Map upstream spellings onto the obfuscated tree's paths."""
    for old, new in (
        ("tikvrpc/", "wirerpc/"),
        ("internal/mockstore/mocktikv/", "internal/mockstore/mockkv/"),
        ("internal/retry/", "internal/client/retry/"),
    ):
        if p.startswith(old):
            p = new + p[len(old):]
    return p


def authored_bank() -> dict[str, dict[str, str]]:
    """file -> {symbol -> unit-name} across all authored repos."""
    out: dict[str, dict[str, str]] = {}
    for cl in ROOT.glob("experiments/pipeline/authored*/*/*/_author/closure.md"):
        unit = cl.parent.parent.name
        repo = cl.parent.parent.parent.name
        text = cl.read_text(errors="replace")
        files = re.findall(r"`?([A-Za-z0-9_/]+\.go)`?", text)
        syms: list[str] = []
        for m in REMOVED_RE.finditer(text):
            syms += re.findall(r"`([A-Za-z_][\w.]*)`", m.group(1))
        # also catch un-backticked comma lists
        for m in REMOVED_RE.finditer(text):
            for tok in re.split(r"[,\s]+", m.group(1)):
                tok = tok.strip("`. ")
                if tok and re.match(r"^[A-Za-z_]\w*(\.\w+)?$", tok):
                    syms.append(tok)
        dest = out.setdefault(f"{repo}", {})
        key = files[0] if files else "?"
        dest.setdefault(key, {})
        for s in set(syms):
            dest[key][s] = unit
    return out


def funcs_in_file(path: Path) -> list[tuple[str, int]]:
    out = []
    for i, line in enumerate(path.read_text(errors="replace").splitlines(), 1):
        m = FUNC_RE.match(line)
        if m:
            recv = m.group(1)
            rname = recv.split()[-1].lstrip("*") if recv else ""
            out.append((f"{rname}.{m.group(2)}" if rname else m.group(2), i))
    return out


def short(sym: str) -> str:
    return sym.split(".")[-1]


def taken_map_for(path: str, bank: dict[str, dict[str, str]]) -> dict[str, str]:
    """All symbols already excised for a given obfuscated relpath."""
    taken: dict[str, str] = {}
    for repo, files in bank.items():
        for f, syms in files.items():
            if f == path or (repo == "client-go" and norm_path(f) == path):
                for s, u in syms.items():
                    taken[s] = f"{repo}/{u}"
    if path in TASK_EXCISIONS:
        for s, u in TASK_EXCISIONS[path].items():
            taken[s] = f"task:{u}"
    # upstream spelling fallback for the static table
    for old, new in (("wirerpc/", "tikvrpc/"), ("internal/mockstore/mockkv/", "internal/mockstore/mocktikv/"), ("internal/client/retry/", "internal/retry/")):
        if path.startswith(old):
            alt = new + path[len(old):]
            for s, u in TASK_EXCISIONS.get(alt, {}).items():
                taken[s] = f"task:{u}"
    return taken


def main(argv: list[str]) -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("args", nargs="*", help="FILE FUNC... or nothing with --spec")
    ap.add_argument("--spec", type=Path, help="excise_spec.json to check")
    ap.add_argument("--funcs", action="store_true", help="list funcs in FILE with TAKEN/FREE")
    ap.add_argument("--tree", type=Path, default=ROOT / "experiments/harbor_nex/base/src")
    a = ap.parse_args(argv)

    bank = authored_bank()
    collisions = 0

    if a.funcs:
        rel = a.args[0]
        taken = taken_map_for(rel, bank)
        for name, line in funcs_in_file(a.tree / rel):
            owner = taken.get(name) or taken.get(short(name))
            mark = f"TAKEN {owner}" if owner else "free"
            print(f"{line:5} {name:55} {mark}")
        if "*" in taken:
            print(f"NOTE: whole file claimed by {taken['*']}")
        return 0

    checks: list[tuple[str, str]] = []
    if a.spec:
        spec = json.loads(a.spec.read_text())
        for unit, files in spec.items():
            for rel, funcs in files.items():
                checks += [(rel, f) for f in funcs]
    elif len(a.args) >= 2:
        checks = [(a.args[0], f) for f in a.args[1:]]
    else:
        ap.error("need --spec or FILE FUNC...")

    for rel, sym in checks:
        taken = taken_map_for(rel, bank)
        owner = taken.get(sym) or taken.get(short(sym)) or ("*" if "*" in taken else None)
        owner = taken.get("*", owner) if owner == "*" else owner
        if owner:
            collisions += 1
            print(f"COLLISION {rel}::{sym} already excised by {owner}")
    print(f"{collisions} collision(s)")
    return 1 if collisions else 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
