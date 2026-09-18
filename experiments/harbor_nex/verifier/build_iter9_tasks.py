"""Build Harbor tasks_iter9 for KeyspaceCodec (subsystem) and MemGet (perf gate)."""

from __future__ import annotations

from pathlib import Path

from openswe_traces.synth.difficulty import name_leakage
from openswe_traces.synth.harbor_tasks import (
    AGENT_TIMEOUT_HARD_SEC,
    build_task,
    file_sha256,
    instruction_self_check,
    render_checksum_guard,
)

ROOT = Path("/home/evan/Documents/open_swe_traces_research")
REPO = ROOT / "experiments/codegraph_bugs/repos/client-go"
BUGS = ROOT / "experiments/codegraph_bugs/bugs/client-go"
OUT = ROOT / "experiments/harbor_nex/tasks_iter9"
BASE = "190f0cce536f835b72481f7bcd0a9448bb1e5202"

KEYSPACE_F2P = ["TestCodecV2", "TestRegionCache"]
KEYSPACE_FILES = [
    "internal/apicodec/codec.go",
    "internal/apicodec/codec_v1.go",
    "internal/apicodec/codec_v2.go",
    "internal/apicodec/mem_codec.go",
    "internal/locate/pd_codec.go",
]
KEYSPACE_SYMBOLS = [
    "ParseKeyspaceID",
    "DecodeKey",
    "EncodeKey",
    "EncodeRequest",
    "DecodeResponse",
    "DecodeBucketKeys",
    "attachAPICtx",
    "IsDecodeError",
    "NewCodecPDClientWithKeyspace",
    "GetKeyspaceID",
    "decodeRegionError",
    "encodeKeys",
    "encodeMutations",
    "decodePairs",
    "NewCodecV2",
    "codecV2",
]

KEYSPACE_BRIEF = """
API v2 multiplexes many logical keyspaces onto one cluster. User keys must be
namespaced with a 4-byte header: a mode byte (raw vs transactional) plus a
24-bit keyspace id. Region boundary keys additionally wrap that header in a
mem-comparable encoding so range order is preserved. Responses and region-error
metadata must strip the header back to user keys. RPC context must carry API
version 2 and the keyspace id, including MPP/compact payloads. Bucket split
keys on a region decode the same way as the region start/end. A malformed
region key is a fatal decode; the client must not retry it with backoff.

A no-op pass-through that leaves user keys unprefixed is wrong: RawGet of
user key `key` in keyspace 0x1092 must wire as bytes
`0x72 0x00 0x10 0x92 0x6b 0x65 0x79`, not the bare user key. Parsing a
prefixed key must yield keyspace id 0x10203, not 0xffffffff. Bucket keys
`a`, `b`, `c` must round-trip to those same user keys.
"""

MEM_F2P = ["TestGetSet", "TestKVGetSet"]
MEM_FILES = ["internal/unionstore/memdb.go"]
MEM_SYMBOLS = ["Get", "set", "traverse", "MemDB", "newMemDB"]

MEM_BRIEF = """
The in-memory write buffer must store key/value pairs and return the latest
value for a key, or report that the key does not exist. Lookups after N
inserts must stay logarithmic in N: a full scan of every stored pair on each
lookup is too slow. On this machine, 5000 sequential lookups after 5000
inserts must stay under the ns/op ceiling recorded from the reference
implementation (gold time x 3). The performance command is:

go test -count=1 -timeout 15m -bench=^BenchmarkGet$ -benchtime=5000x ./internal/unionstore/
"""


def _append_checksum(test_sh: Path, rel: str, digest: str) -> None:
    text = test_sh.read_text()
    needle = "if go test"
    extra = render_checksum_guard([(rel, digest)])
    # render_checksum_guard includes the helper; keep only the echo line.
    echo_line = [ln for ln in extra.splitlines() if ln.startswith("echo ")][-1]
    if echo_line in text:
        return
    if needle not in text:
        raise SystemExit(f"no go test line in {test_sh}")
    text = text.replace(needle, echo_line + "\n" + needle, 1)
    test_sh.write_text(text)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    api_out = Path("/tmp/iter9a_api.txt").read_text(errors="replace")
    loc_out = Path("/tmp/iter9a_loc.txt").read_text(errors="replace")
    mem_out = Path("/tmp/iter9b_f2p.txt").read_text(errors="replace")

    ks = build_task(
        REPO,
        BASE,
        BUGS / "KeyspaceCodec.patch",
        KEYSPACE_F2P,
        OUT / "client-go-keyspacecodec",
        test_output=api_out + "\n" + loc_out,
        agent_timeout_sec=AGENT_TIMEOUT_HARD_SEC,
        user_context=KEYSPACE_BRIEF,
        extra_redact=KEYSPACE_SYMBOLS + [Path(f).name for f in KEYSPACE_FILES],
        checksum_test_files=True,
        locality=2,
        reproduce_command="go test -count=1 -timeout 15m ./internal/apicodec/ ./internal/locate/",
        kind="feature",
    )
    mem_limit = 754.0  # docker gold 251.4 ns/op * 3
    mg = build_task(
        REPO,
        BASE,
        BUGS / "MemGet.patch",
        MEM_F2P,
        OUT / "client-go-memget",
        test_output=mem_out,
        agent_timeout_sec=AGENT_TIMEOUT_HARD_SEC,
        user_context=MEM_BRIEF,
        extra_redact=MEM_SYMBOLS + ["memdb.go"],
        checksum_test_files=True,
        locality=2,
        reproduce_command="go test -count=1 -timeout 15m ./internal/unionstore/",
        kind="feature",
        perf_bench="BenchmarkGet",
        perf_limit_ns=mem_limit,
        perf_benchtime="5000x",
    )
    bench = mg / "environment" / "src" / "internal/unionstore/memdb_bench_test.go"
    _append_checksum(mg / "tests" / "test.sh", "internal/unionstore/memdb_bench_test.go", file_sha256(bench))

    for name, task, f2p, symbols, files, output in (
        ("keyspacecodec", ks, KEYSPACE_F2P, KEYSPACE_SYMBOLS, KEYSPACE_FILES, api_out + loc_out),
        ("memget", mg, MEM_F2P, MEM_SYMBOLS, MEM_FILES, mem_out),
    ):
        instr = (task / "instruction.md").read_text()
        sh = (task / "tests" / "test.sh").read_text()
        chk = instruction_self_check(
            instr,
            test_sh=sh,
            f2p_tests=f2p,
            changed_symbols=symbols,
            changed_files=files,
            locality=2,
            packages=["internal/apicodec", "internal/locate"]
            if name == "keyspacecodec"
            else ["internal/unionstore"],
            reproduce_command="go test"
            + (
                " ./internal/apicodec/"
                if name == "keyspacecodec"
                else " ./internal/unionstore/"
            ),
        )
        leak = name_leakage(f2p, instr, changed_symbols=symbols, changed_files=files)
        print(name, "instruction_ok", chk["ok"], "leak_ok", leak["ok"])
        print("  leaked_symbols", chk["leaked_symbols"], leak["leaked_symbols"])
        print("  leaked_files", chk["leaked_files"], leak["leaked_files"])
        if not chk["ok"] or not leak["ok"]:
            print(instr[:1200])
            raise SystemExit(f"{name} instruction failed self-check")
    print("built", ks, mg)


if __name__ == "__main__":
    main()
