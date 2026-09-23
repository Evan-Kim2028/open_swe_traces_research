#!/usr/bin/env python3
"""Move Grok-authored units out of their scratch dir into the authored tree.

Grok cannot run in a git worktree (it hangs), so it authors in a plain directory that
nothing else knows about. Without this step its units are exactly as invisible as the 74
that sat unstageable in Devin worktrees for hours - the same failure, a different engine.

Lands them at experiments/pipeline/authored_grok<repo>/<repo>/<unit>/ so that
pipeline_autogen's census (authored_batch*/... glob is authored*) sees them and queues
verification without anyone asking.

  grok_harvest.py [repo ...] [--apply]
"""
import os, sys, glob, shutil, json

R = "/home/evan/Documents/open_swe_traces_research"
TMP = "/home/evan/.claude/jobs/a1eaeb86/tmp"
APPLY = "--apply" in sys.argv
REPOS = [a for a in sys.argv[1:] if not a.startswith("--")]

# what a unit needs before verification can do anything with it
NEED = ("_author/gold.patch", "_author/bugreport.md", "_author/DETAILS.md")


def main():
    scratches = sorted(glob.glob(f"{TMP}/grokauth_*"))
    if REPOS:
        scratches = [s for s in scratches
                     if os.path.basename(s).replace("grokauth_", "") in REPOS]
    if not scratches:
        print("no grok scratch dirs")
        return 1
    total = moved = skipped = 0
    for s in scratches:
        repo = os.path.basename(s).replace("grokauth_", "")
        dest_root = f"{R}/experiments/pipeline/authored_grok{repo.replace('-','')}/{repo}"
        units = sorted(glob.glob(f"{s}/units/*/"))
        print(f"{repo}: {len(units)} unit(s) in scratch")
        for u in units:
            total += 1
            name = os.path.basename(u.rstrip("/"))
            missing = [n for n in NEED
                       if not (os.path.exists(os.path.join(u, n))
                               and os.path.getsize(os.path.join(u, n)) > 0)]
            if missing:
                # An incomplete unit is not an asset. Leaving it in the tree means the
                # census counts it as authored and it blocks the authoring gate forever
                # while never being verifiable.
                print(f"  SKIP {name}: missing {', '.join(m.split('/')[-1] for m in missing)}")
                skipped += 1
                continue
            dst = os.path.join(dest_root, name)
            if os.path.exists(dst):
                print(f"  exists {name}")
                continue
            if APPLY:
                os.makedirs(dest_root, exist_ok=True)
                shutil.copytree(u.rstrip("/"), dst)
            moved += 1
        rep = f"{s}/outputs/grokauth_{repo}.md"
        if os.path.exists(rep) and APPLY:
            os.makedirs(f"{R}/analytics/research", exist_ok=True)
            shutil.copy2(rep, f"{R}/analytics/research/grokauth_{repo}.md")
            print(f"  report -> analytics/research/grokauth_{repo}.md")
    print(f"\n{'harvested' if APPLY else 'would harvest'} {moved} of {total} unit(s); "
          f"{skipped} incomplete")
    if not APPLY:
        print("dry run — pass --apply")
    return 0


def cli():
    sys.exit(main())


if __name__ == "__main__":
    cli()
