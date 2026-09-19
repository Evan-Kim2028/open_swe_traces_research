#!/usr/bin/env bash
# Rebuild ladder-base images on a current Go toolchain and record offline test baselines. 2 builds at a time, niced.
P=/home/evan/Documents/open_swe_traces_research/experiments/pipeline
IMG=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep -E '^golang:(1\.27|latest)$' | head -1)
build_one() {
  r=$1; d=$P/repos/$r; [ -d $d/src ] || { echo "$r: no src"; return; }
  sed -i "s|^FROM golang:.*|FROM $IMG|" $d/Dockerfile; grep -q "GOTOOLCHAIN" $d/Dockerfile || sed -i "s|^FROM .*|&\nENV GOTOOLCHAIN=auto GOFLAGS=-mod=mod|" $d/Dockerfile
  if nice -n 10 docker build -q --cpu-quota=400000 -t ladder-base:$r $d > $d/docker_build.log 2>&1; then
    nice -n 10 docker run --rm --network none --cpus=4 --memory=8g ladder-base:$r bash -c 'cd /app && go test ./... -count=1 -timeout 20m 2>&1' > $d/test.jsonl 2>&1
    pass=$(grep -cE "^ok\s" $d/test.jsonl); fail=$(grep -cE "^(FAIL|---)\s" $d/test.jsonl)
    python3 -c "import json;json.dump({'build_ok':True,'packages_passing':$pass,'packages_failing':$fail,'image':'ladder-base:$r','go_image':'$IMG'},open('$d/baseline.json','w'))"
    echo "$(date -u +%H:%M) $r: build ok, packages passing=$pass failing=$fail"
  else
    python3 -c "import json;json.dump({'build_ok':False,'image':'ladder-base:$r','go_image':'$IMG'},open('$d/baseline.json','w'))"; echo "$(date -u +%H:%M) $r: BUILD FAILED ($(grep -m1 -oE 'error.{0,120}' $d/docker_build.log))"
  fi
}
for r in argo kops knative-client go-github goa nats-server; do build_one $r; done
python3 - <<'PY'
import json,os,glob
P='/home/evan/Documents/open_swe_traces_research/experiments/pipeline'
rows=[]
for d in sorted(glob.glob(P+'/repos/*/baseline.json')):
    r=os.path.basename(os.path.dirname(d)); b=json.load(open(d))
    if b.get('build_ok') and b.get('packages_passing',0)>=5: rows.append((r,b['packages_passing']))
with open(P+'/repos.yaml','w') as f:
    f.write('repos:\n')
    for r,n in rows: f.write(f"  - name: {r}\n    src: experiments/pipeline/repos/{r}/src\n    base_image: ladder-base:{r}\n    packages_passing: {n}\n    language: go\n")
print('repos.yaml:',rows)
PY
echo "$(date -u +%H:%M) rebuild done"
