#!/usr/bin/env bash
# runs briefs two at a time on or-grunt; waits for each pair
D=/home/evan/devin-tasks; W=/home/evan/Documents/lake-of-rage
run_pair(){ for n in "$@"; do : > $D/f$n.out; /home/evan/.local/bin/or-grunt $D/f$n.md $W $D/f$n.out; done
  for n in "$@"; do until grep -q '^exit' $D/f$n.out 2>/dev/null; do sleep 30; done; echo "$(date -Is) f$n done: $(grep -oE 'https://github.com/[^ )]+/(pull|issues)/[0-9]+' $D/f$n.out | tail -1)"; done; }
run_pair 2965 2967
run_pair 2968 2827
run_pair 2826 2828
echo "$(date -Is) ALL DONE"
