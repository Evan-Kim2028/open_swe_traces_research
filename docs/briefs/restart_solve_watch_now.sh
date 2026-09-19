#!/bin/bash
D=/home/evan/devin-tasks
w="solve""-watch --interval"; r="restart_watch""_at_boundary"
pkill -f "bash $D/$r"
for p in $(pgrep -f "$w"); do kill $p; done
sleep 3
bash $D/start_solve_watch.sh
