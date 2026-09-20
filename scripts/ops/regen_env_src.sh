#!/usr/bin/env bash
# Regenerate a batch's environment/src trees after a disk cleanup deleted them.
#
# environment/src is the EXCISED tree (the bug already introduced), and the base image holds
# the PRISTINE tree. gold.patch turns excised into pristine, so reverse-applying gold to a
# copy of the image's /app reproduces the excised tree exactly. No network, no re-clone.
#
# Usage: regen_env_src.sh <ladder-base image> <batch dir containing <unit>/ dirs>
set -u
IMG="${1:?image}"; BATCH="${2:?batch dir}"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
C=$(docker create "$IMG") || exit 1
docker cp "$C:/app/." "$TMP/pristine/" >/dev/null 2>&1 || { docker rm "$C" >/dev/null; exit 1; }
docker rm "$C" >/dev/null
ok=0; skip=0; fail=0
for u in "$BATCH"/*/; do
  [ -d "$u" ] || continue
  case "$(basename "$u")" in _*) continue;; esac
  [ -f "$u/patches/gold.patch" ] || { skip=$((skip+1)); continue; }
  if [ -d "$u/environment/src" ] && [ "$(ls -A "$u/environment/src" 2>/dev/null | wc -l)" -gt 0 ]; then
    skip=$((skip+1)); continue
  fi
  rm -rf "$u/environment/src"
  cp -a "$TMP/pristine" "$u/environment/src"
  if ( cd "$u/environment/src" && patch -p1 -R --silent < "$OLDPWD/$u/patches/gold.patch" ) 2>/dev/null \
     || ( cd "$u/environment/src" && patch -p1 -R --silent < "$(readlink -f "$u/patches/gold.patch")" ) 2>/dev/null; then
    ok=$((ok+1))
  else
    fail=$((fail+1)); echo "FAILED: $(basename "$u")"
  fi
done
echo "regenerated=$ok skipped=$skip failed=$fail"
