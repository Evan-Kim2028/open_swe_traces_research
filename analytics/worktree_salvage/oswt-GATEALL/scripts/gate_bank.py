"""CLI wrapper for openswe_traces.bank_gates — see that module's docstring."""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "src"))

from openswe_traces.bank_gates import main

if __name__ == "__main__":
    raise SystemExit(main())
