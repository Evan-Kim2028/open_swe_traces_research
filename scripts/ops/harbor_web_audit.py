#!/usr/bin/env python3
"""Audit Harbor cursor-cli trials for web tool use; print per-task verdicts. Usage: harbor_web_audit.py <jobdir> [<jobdir>...]"""
import glob,os,re,json,sys,datetime as dt
rows=[]
for J in sys.argv[1:]:
    for t in sorted(glob.glob(J.rstrip('/')+'/*/')):
        if not os.path.isdir(t+'agent'): continue
        txt=''
        for f in glob.glob(t+'agent/*'):
            try: txt+=open(f,errors='ignore').read()
            except Exception: pass
        wf=len(re.findall(r'webFetchToolCall',txt)); ws=len(re.findall(r'webSearchToolCall',txt))
        r={}; 
        try: r=json.load(open(t+'result.json'))
        except Exception: pass
        rew=(r.get('verifier_result') or {}).get('rewards',{}).get('reward')
        try: mins=round((dt.datetime.fromisoformat(r['finished_at'])-dt.datetime.fromisoformat(r['started_at'])).total_seconds()/60,1)
        except Exception: mins=None
        name=os.path.basename(t.rstrip('/')).split('__')[0]
        verdict='CONTAMINATED' if (wf or ws) else ('PASS' if rew==1.0 else ('FAIL' if rew==0.0 else 'PENDING'))
        rows.append((os.path.basename(J.rstrip('/')),name,verdict,rew,mins,wf,ws))
print(f"{'job':22} {'task':38} {'verdict':13} rew  min  fetch search")
for r in rows: print(f"{r[0]:22} {r[1]:38} {r[2]:13} {str(r[3]):4} {str(r[4]):5} {r[5]:5} {r[6]:6}")
from collections import Counter; print(Counter(r[2] for r in rows))
