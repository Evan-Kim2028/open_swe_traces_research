#!/usr/bin/env python3
"""Import every moved module in its own process and report failures or import-time output.

A module that prints, raises or hangs on import would break every script that imports it
through its shim. Run with the cwd the modules normally see (a checkout or a
ladder_golden snapshot), since some read cwd-relative files at import.

Usage: import_check.py <cwd>   (run from the repo root)
"""
import os
import pathlib
import subprocess
import sys

sys.path.insert(0, "scripts/dev")
from move_module import MOVES  # noqa: E402

SRC = str(pathlib.Path("src").resolve())


def main(cwd):
    env = dict(os.environ, PYTHONPATH=SRC)
    bad = 0
    for mod in MOVES.values():
        try:
            p = subprocess.run([sys.executable, "-c", f"import {mod}"], cwd=cwd, env=env,
                               capture_output=True, text=True, timeout=120)
        except subprocess.TimeoutExpired:
            print(f"HANG   {mod}")
            bad += 1
            continue
        out = (p.stdout + p.stderr).strip()
        if p.returncode:
            print(f"FAIL   {mod}: {out.splitlines()[-1] if out else p.returncode}")
            bad += 1
        elif out:
            print(f"OUTPUT {mod}: {out.splitlines()[0][:120]}")
    print(f"{len(MOVES)} modules, {bad} failed")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1] if len(sys.argv) > 1 else "."))
