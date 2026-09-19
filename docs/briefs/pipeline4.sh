#!/usr/bin/env bash
# waits for a free slot (≤2 or-grunt runners), then launches f-readability and x-xref-on
D=/home/evan/devin-tasks; W=/home/evan/Documents/lake-of-rage; c="/tmp/or-gr"; d="unt"
slot(){ while [ "$(pgrep -f "python $c$d" | wc -l)" -ge 2 ]; do sleep 60; done; }
log(){ echo "$(date -Is) $*"; }
slot; log "START f-readability"; : > $D/f-readability.out; /home/evan/.local/bin/or-grunt $D/f-readability.md $W $D/f-readability.out; sleep 5
slot; log "START x-xref-on"; : > $D/x-xref-on.out; /home/evan/.local/bin/or-grunt $D/x-xref-on.md $W $D/x-xref-on.out
log "QUEUE LAUNCHED"
