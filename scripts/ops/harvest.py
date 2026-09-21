#!/usr/bin/env python3
"""Copy finished verifier/reconciler output out of Devin worktrees into the main checkout.

This step did not exist, and its absence was the real reason the dataset went flat. A VF job
writes `tests/` into its OWN worktree; a RC job writes `_author/contract.md` into its own.
Nothing ever moved them. census() takes the max across roots, so autogen saw "20/20 go-git
verified" and never re-queued, while stage_units.py reads only the main checkout and saw a
unit with no suite and no contract - unstageable at any rung. 74 units sat finished and
invisible while seven more authoring jobs queued behind them.

Copy only artifacts, never source: a unit's `tests/` tree and `_author/contract.md`. Never
overwrite an artifact the main checkout already has - the main copy is the one that has been
trialled, and a worktree may hold an older or partial attempt.

Usage: harvest.py [--apply]
"""
import os, glob, shutil, sys

R = "/home/evan/Documents/open_swe_traces_research"
APPLY = "--apply" in sys.argv


def unit_dirs(root):
    # authored* not authored_batch*: the grok batches (authored_grokgogit,
    # authored_groknatsserver) and the per-repo au5 batches carry no "batch" in the
    # name, so a batch-only glob silently skipped them — 10 fully verified units sat
    # unharvested with harvest reporting "0 recoverable". Same bug as autogen's census.
    for d in glob.glob(f"{root}/experiments/pipeline/authored*/*/*/"):
        if os.path.basename(os.path.dirname(os.path.dirname(d.rstrip("/")))).startswith("_"):
            continue
        yield d


def rel(d, root):
    return os.path.relpath(d.rstrip("/"), root)


def has_suite(d):
    return bool(glob.glob(d + "tests/**/*_test.go", recursive=True))


# What stage_units.discover_units() actually demands, plus the artifacts the rungs read.
# "Has a suite" was too weak a definition of complete: a unit with tests/hidden but no
# tests/test.sh passed this check and was never flagged, while the stager rejected it.
REQUIRED = ("tests/test.sh", "_author/contract.md", "_author/bugreport.md",
            "_author/cheat.patch", "_author/gold.patch")


def missing_parts(d):
    miss = [r for r in REQUIRED if not os.path.exists(os.path.join(d, r))]
    if not has_suite(d):
        miss.append("tests/hidden/*_test.go")
    return miss


def main():
    # what the main checkout is missing
    want = {}
    for d in unit_dirs(R):
        k = rel(d, R)
        if not os.path.isdir(os.path.join(d, "_author")):
            continue
        miss = missing_parts(d)
        if miss:
            want[k] = miss
    if not want:
        print("harvest: main checkout has every suite and contract")
        return 0

    # Draw from EVERY root, not one chosen donor. The artifacts for a single unit are
    # split across worktrees - AU2gogithub holds the _author files (bugreport, cheat) and
    # VFbatch3gogithubb holds the suite and test.sh - so picking the donor that supplied
    # the most still left each unit incomplete and unstageable.
    plan = {}
    for root in sorted(glob.glob("/home/evan/Documents/oswt-*")):
        for d in unit_dirs(root):
            k = rel(d, root)
            if k not in want:
                continue
            supplies = [m for m in want[k]
                        if (m.endswith("*_test.go") and has_suite(d))
                        or (not m.endswith("*_test.go")
                            and os.path.exists(os.path.join(d, m)))]
            if supplies:
                plan.setdefault(k, []).append((d, supplies, os.path.basename(root)))

    covered = {k: sorted({m for _, sup, _ in v for m in sup}) for k, v in plan.items()}
    nparts = sum(len(v) for v in covered.values())
    full = sum(1 for k, v in covered.items() if len(v) == len(want[k]))
    print(f"harvest: {len(want)} incomplete unit(s) in main; {len(plan)} recoverable "
          f"({nparts} missing part(s); {full} become fully complete)")

    by_donor = {}
    for k, v in sorted(plan.items()):
        for _, _, donor in v:
            by_donor.setdefault(donor, set()).add(k)
    for donor, ks in sorted(by_donor.items(), key=lambda x: -len(x[1])):
        print(f"  {donor:28s} {len(ks):3d} unit(s)  e.g. {sorted(ks)[0]}")

    if not APPLY:
        print("\ndry run — pass --apply to copy")
        return 0

    ok = fail = 0
    for k, donors in sorted(plan.items()):
        dst = os.path.join(R, k)
        copied = 0
        for src, _, _ in donors:
            try:
                # Fill in missing FILES, never whole directories, and never overwrite.
                # Copying tests/ as a tree only when the destination lacked it meant a
                # unit that already had tests/hidden could never receive anything later:
                # harvest ran at 05:40 and the verifier wrote test.sh at 05:55, so twelve
                # go-github units had a suite but no runner and the stager rejected every
                # one with "no authored+verified units discovered". Not overwriting keeps
                # the already-trialled copy authoritative.
                for sub in ("tests", "_author"):
                    sdir = os.path.join(src, sub)
                    if not os.path.isdir(sdir):
                        continue
                    for root_dir, _, files in os.walk(sdir):
                        rel_dir = os.path.relpath(root_dir, src)
                        for fn in files:
                            df = os.path.join(dst, rel_dir, fn)
                            if os.path.exists(df):
                                continue
                            os.makedirs(os.path.dirname(df), exist_ok=True)
                            shutil.copy2(os.path.join(root_dir, fn), df)
                            copied += 1
            except Exception as e:
                print(f"  FAIL {k} from {src}: {e}")
                fail += 1
        if copied:
            ok += 1
    print(f"\nharvested {ok} unit(s), {fail} failure(s)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
