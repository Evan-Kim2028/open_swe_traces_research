#!/usr/bin/env bash
D=/home/evan/devin-tasks; c="/tmp/or-gr"; d="unt"
while [ "$(pgrep -f "python $c$d" | wc -l)" -ge 2 ]; do sleep 60; done
echo "$(date -Is) START f2994b (silphcoanalytics)"; : > $D/f2994b.out; /home/evan/.local/bin/or-grunt $D/f2994b.md /home/evan/Documents/silphcoanalytics $D/f2994b.out
