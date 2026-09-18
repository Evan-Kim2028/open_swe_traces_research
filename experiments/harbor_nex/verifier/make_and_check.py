"""Generate alt/cheat patches on buggy trees and run f2p checks."""

from __future__ import annotations

import os
import shutil
import subprocess
import tempfile
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

from openswe_traces.synth.harbor_tasks import snapshot_bugged_tree

ROOT = Path("/home/evan/Documents/open_swe_traces_research")
REPO = ROOT / "experiments/codegraph_bugs/repos/client-go"
BUGS = ROOT / "experiments/codegraph_bugs/bugs/client-go"
BASE = "190f0cce536f835b72481f7bcd0a9448bb1e5202"
OUT = ROOT / "experiments/harbor_nex/verifier"
GO = "/usr/local/go/bin/go"

# (symbol, f2p tests, packages, relpath, buggy_needle, alt_text, cheat_text)
SPECS: list[tuple] = [
    (
        "NewRequest",
        ["TestRegionRequestToThreeStores"],
        ["internal/locate"],
        "tikvrpc/tikvrpc.go",
        """func NewRequest(typ CmdType, pointer interface{}, ctxs ...kvrpcpb.Context) *Request {
	return &Request{
		Type: typ,
		Req:  pointer,
	}
}""",
        """func NewRequest(typ CmdType, pointer interface{}, ctxs ...kvrpcpb.Context) *Request {
	req := &Request{Type: typ, Req: pointer}
	if len(ctxs) > 0 {
		req.Context = ctxs[0]
	}
	return req
}""",
        """func NewRequest(typ CmdType, pointer interface{}, ctxs ...kvrpcpb.Context) *Request {
	if len(ctxs) > 0 && typ == CmdEmpty {
		return &Request{Type: typ, Req: pointer, Context: ctxs[0]}
	}
	return &Request{
		Type: typ,
		Req:  pointer,
	}
}""",
    ),
    (
        "NewRegionRequestSender",
        ["TestRegionRequestToThreeStores"],
        ["internal/locate"],
        "internal/locate/region_request.go",
        """	return &RegionRequestSender{
		regionCache: regionCache,
		apiVersion:  regionCache.codec.GetAPIVersion(),
		client:      nil,
	}""",
        """	s := &RegionRequestSender{
		regionCache: regionCache,
		apiVersion:  regionCache.codec.GetAPIVersion(),
	}
	s.client = client
	return s""",
        """	if regionCache == nil {
		return &RegionRequestSender{client: client}
	}
	return &RegionRequestSender{
		regionCache: regionCache,
		apiVersion:  regionCache.codec.GetAPIVersion(),
		client:      nil,
	}""",
    ),
    (
        "GetGlobalConfig",
        [
            "TestPanicInRecvLoop",
            "TestRecvErrorInMultipleRecvLoops",
            "TestRegionCache",
            "TestRawKV",
        ],
        ["internal/client", "internal/locate", "rawkv"],
        "config/config.go",
        """func GetGlobalConfig() *Config {
	return &Config{}
}""",
        """func GetGlobalConfig() *Config {
	v := globalConf.Load()
	if cfg, ok := v.(*Config); ok {
		return cfg
	}
	cfg := DefaultConfig()
	return &cfg
}""",
        """func GetGlobalConfig() *Config {
	cfg := globalConf.Load().(*Config)
	if cfg.Path == "__never_used_path__" {
		return cfg
	}
	return &Config{}
}""",
    ),
    (
        "IsErrNotFound",
        [
            "TestPipelinedFlushGet",
            "TestUnionStoreGetSet",
            "TestUnionStoreDelete",
            "TestBufferBatchGetter",
        ],
        ["internal/unionstore", "txnkv/transaction"],
        "error/error.go",
        """func IsErrNotFound(err error) bool {
	return !errors.Is(err, ErrNotExist)
}""",
        """func IsErrNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrNotExist)
}""",
        """func IsErrNotFound(err error) bool {
	if err != nil && err.Error() == "not exist from test-only sentinel" {
		return true
	}
	return !errors.Is(err, ErrNotExist)
}""",
    ),
    (
        "IsFakeRegionError",
        ["TestRegionRequestToThreeStores", "TestRegionRequestToSingleStore"],
        ["internal/locate"],
        "internal/locate/region_request.go",
        """func IsFakeRegionError(err *errorpb.Error) bool {
	return err != nil && err.GetEpochNotMatch() != nil && len(err.GetEpochNotMatch().CurrentRegions) != 0
}""",
        """func IsFakeRegionError(err *errorpb.Error) bool {
	if err == nil {
		return false
	}
	enm := err.GetEpochNotMatch()
	if enm == nil {
		return false
	}
	return len(enm.CurrentRegions) == 0
}""",
        """func IsFakeRegionError(err *errorpb.Error) bool {
	if err != nil && err.GetNotLeader() != nil {
		return true
	}
	return err != nil && err.GetEpochNotMatch() != nil && len(err.GetEpochNotMatch().CurrentRegions) != 0
}""",
    ),
]


def unified_from_edit(buggy_text: str, new_text: str, rel: str) -> str:
    with tempfile.TemporaryDirectory() as tmp:
        a = Path(tmp) / "a"
        b = Path(tmp) / "b"
        (a / rel).parent.mkdir(parents=True)
        (b / rel).parent.mkdir(parents=True)
        (a / rel).write_text(buggy_text)
        (b / rel).write_text(new_text)
        proc = subprocess.run(
            ["diff", "-u", f"a/{rel}", f"b/{rel}"],
            cwd=tmp,
            capture_output=True,
            text=True,
            check=False,
        )
        out = proc.stdout
    # rewrite headers to git-style
    lines = out.splitlines(keepends=True)
    body = []
    for ln in lines:
        if ln.startswith("--- "):
            body.append(f"diff --git a/{rel} b/{rel}\n")
            body.append(f"--- a/{rel}\n")
        elif ln.startswith("+++ "):
            body.append(f"+++ b/{rel}\n")
        else:
            body.append(ln)
    return "".join(body)


def go_test(tree: Path, tests: list[str], packages: list[str], timeout: int = 600) -> tuple[bool, str]:
    pattern = "^(" + "|".join(tests) + ")$"
    args = [GO, "test", "-count=1", "-timeout", "14m", "-run", pattern]
    args.extend("./" + p + "/..." for p in packages)
    env = {**os.environ, "PATH": "/usr/local/go/bin:" + os.environ.get("PATH", "")}
    proc = subprocess.run(
        args, cwd=tree, capture_output=True, timeout=timeout, env=env, check=False
    )
    out = (proc.stdout + proc.stderr).decode("utf-8", "replace")
    return proc.returncode == 0, out


def _check_one(spec: tuple) -> tuple[str, str, str, bool, str]:
    symbol, tests, packages, rel, needle, alt, cheat = spec
    tmp = tempfile.mkdtemp(prefix=f"ver_{symbol}_")
    try:
        tree = Path(tmp) / "src"
        snapshot_bugged_tree(REPO, BASE, BUGS / f"{symbol}.patch", tree)
        target = tree / rel
        text = target.read_text()
        if needle not in text:
            raise RuntimeError(f"{symbol}: buggy needle not found in {rel}")
        alt_text = text.replace(needle, alt, 1)
        cheat_text = text.replace(needle, cheat, 1)
        (BUGS / f"{symbol}.alt.patch").write_text(unified_from_edit(text, alt_text, rel))
        (BUGS / f"{symbol}.cheat.patch").write_text(unified_from_edit(text, cheat_text, rel))
        alt_tree = Path(tmp) / "alt"
        cheat_tree = Path(tmp) / "cheat"
        shutil.copytree(tree, alt_tree)
        shutil.copytree(tree, cheat_tree)
        (alt_tree / rel).write_text(alt_text)
        (cheat_tree / rel).write_text(cheat_text)
        with ThreadPoolExecutor(max_workers=2) as inner:
            fut_alt = inner.submit(go_test, alt_tree, tests, packages)
            fut_cheat = inner.submit(go_test, cheat_tree, tests, packages)
            alt_ok, alt_out = fut_alt.result()
            cheat_ok, cheat_out = fut_cheat.result()
        (OUT / f"{symbol}.alt.log").write_text(alt_out[-12000:])
        (OUT / f"{symbol}.cheat.log").write_text(cheat_out[-12000:])
        extra = ""
        if not alt_ok:
            extra += "ALT FAIL\n" + alt_out[-2000:]
        if cheat_ok:
            extra += "CHEAT PASS\n" + cheat_out[-2000:]
        alt_verdict = "accept" if alt_ok else "REJECT"
        cheat_verdict = "reject" if not cheat_ok else "ACCEPT (bad)"
        return symbol, alt_verdict, cheat_verdict, alt_ok and not cheat_ok, extra
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


def main() -> int:
    os.environ["PATH"] = "/usr/local/go/bin:" + os.environ.get("PATH", "")
    OUT.mkdir(parents=True, exist_ok=True)
    rows: list[tuple[str, str, str, bool]] = []
    with ThreadPoolExecutor(max_workers=3) as pool:
        futs = {pool.submit(_check_one, spec): spec[0] for spec in SPECS}
        for fut in as_completed(futs):
            symbol, alt_v, cheat_v, survive, extra = fut.result()
            print(f"=== {symbol} alt={alt_v} cheat={cheat_v} survive={survive} ===", flush=True)
            if extra:
                print(extra, flush=True)
            rows.append((symbol, alt_v, cheat_v, survive))
    rows.sort()
    summary = ["symbol,alt,cheat,survive"]
    for r in rows:
        summary.append(",".join(str(x) for x in r))
    (OUT / "table.csv").write_text("\n".join(summary) + "\n")
    print("\n".join(summary))
    return 0 if all(r[3] for r in rows) else 1


if __name__ == "__main__":
    raise SystemExit(main())
