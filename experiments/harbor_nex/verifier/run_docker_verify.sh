#!/usr/bin/env bash
set -euo pipefail
cd /home/evan/Documents/open_swe_traces_research
LOG=experiments/harbor_nex/verifier/docker_verify.log
echo "===== START $(date -Is) =====" >> "$LOG"
verify_one() {
  echo "===== $1 =====" | tee -a "$LOG"
  if bash experiments/harbor_nex/verifier/docker_verify.sh "$@" >>"$LOG" 2>&1; then
    echo "OK $1" | tee -a "$LOG"
  else
    echo "FAIL $1" | tee -a "$LOG"
    return 1
  fi
}
# daily (twosumbest rebuilt)
verify_one experiments/harbor_nex/tasks/dailycodingproblem-go-twosumbest harbor-nex-twosumbest \
  experiments/codegraph_bugs/repos/dailycodingproblem-go day1/problem.go
# client-go
verify_one experiments/harbor_nex/tasks/client-go-iserrnotfound harbor-nex-iserrnotfound \
  experiments/codegraph_bugs/repos/client-go error/error.go
verify_one experiments/harbor_nex/tasks/client-go-newrequest harbor-nex-newrequest \
  experiments/codegraph_bugs/repos/client-go tikvrpc/tikvrpc.go
verify_one experiments/harbor_nex/tasks/client-go-newregionrequestsender harbor-nex-newregionrequestsender \
  experiments/codegraph_bugs/repos/client-go internal/locate/region_request.go
verify_one experiments/harbor_nex/tasks/client-go-isfakeregionerror harbor-nex-isfakeregionerror \
  experiments/codegraph_bugs/repos/client-go internal/locate/region_request.go
verify_one experiments/harbor_nex/tasks/client-go-newbackofferwithvars harbor-nex-newbackofferwithvars \
  experiments/codegraph_bugs/repos/client-go config/retry/backoff.go
verify_one experiments/harbor_nex/tasks/client-go-getglobalconfig harbor-nex-getglobalconfig \
  experiments/codegraph_bugs/repos/client-go config/config.go
echo ALL_DOCKER_VERIFY_DONE | tee -a "$LOG"
