#!/usr/bin/env python3
"""Sum Devin token usage from a Devin CLI sessions.db (message_nodes.metadata / chat_message JSON).
Usage: devin_tokens.py <sessions.db> [session_id]"""
import json,re,sqlite3,sys
def walk(o,acc):
    if isinstance(o,dict):
        for k,v in o.items():
            kl=k.lower()
            if kl in('input_tokens','prompt_tokens'): acc['in']+=int(v or 0)
            elif kl in('output_tokens','completion_tokens'): acc['out']+=int(v or 0)
            elif 'cache' in kl and 'token' in kl and isinstance(v,(int,float)): acc['cache']+=int(v)
            else: walk(v,acc)
    elif isinstance(o,list):
        for v in o: walk(v,acc)
def main(db,sid=None):
    c=sqlite3.connect(db); tot={}
    q="select session_id, chat_message, metadata from message_nodes"+(" where session_id=?" if sid else "")
    for s,cm,md in c.execute(q,(sid,) if sid else ()):
        acc=tot.setdefault(s,{'in':0,'out':0,'cache':0,'msgs':0}); acc['msgs']+=1
        for blob in (cm,md):
            if not blob: continue
            try: walk(json.loads(blob),acc)
            except Exception:
                for m in re.finditer(r'"(input|prompt)_tokens"\s*:\s*(\d+)',str(blob)): acc['in']+=int(m.group(2))
                for m in re.finditer(r'"(output|completion)_tokens"\s*:\s*(\d+)',str(blob)): acc['out']+=int(m.group(2))
    for s,a in tot.items(): print(json.dumps({'session':s,**a}))
if __name__=='__main__': main(sys.argv[1], sys.argv[2] if len(sys.argv)>2 else None)
