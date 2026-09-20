#!/usr/bin/env bash
# The grok.com OAuth session token is accepted by api.x.ai as a bearer, so harbor's
# grok-build agent works without a separate xAI key. expires_at is an ISO-8601 string,
# not an epoch — parsing it as a float silently emptied the key and cost a smoke test.
python3 -c "
import json,sys,datetime
d=json.load(open('/home/evan/.grok/auth.json'))
v=list(d.values())[0]
exp=v.get('expires_at')
if exp:
    try:
        e=datetime.datetime.fromisoformat(str(exp).replace('Z','+00:00'))
        if e <= datetime.datetime.now(datetime.timezone.utc):
            sys.stderr.write('grok token EXPIRED at %s — run: grok login\n'%exp); sys.exit(1)
    except ValueError:
        pass
print(v['key'])
"
