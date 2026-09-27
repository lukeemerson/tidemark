#!/usr/bin/env zsh
# capture.zsh <draft number> <lazy|live> <cols> <rows> -> render/NN-state.html fragment
D=~/dev/terminal-configs/monitor-drafts
T=${0:A:h}
n=${(l:2::0:)1} state=$2 cols=$3 rows=$4
raw=$T/$n-$state-$cols.raw
if [[ $state == lazy ]]; then feed='sleep 31.7'; wait=3; else feed="cat $T/mactop.raw; sleep 31.7"; wait=12; fi
rm -f $raw
script -q $raw zsh -c "stty rows $rows cols $cols; MONITOR_FEED='$feed' zsh $D/v$n.zsh" </dev/null >/dev/null 2>&1 &
local p=$!
sleep $wait
pkill -f "v$n.zsh" 2>/dev/null
pkill -f 'sleep 31.7' 2>/dev/null
wait $p 2>/dev/null # let script flush after the draft exits
# drop everything from the final cleanup (alt-screen exit) onward
python3 - $raw <<'EOF'
import sys
p = sys.argv[1]; d = open(p, 'rb').read()
i = d.rfind(b'\x1b[?1049l')
if i > 0: d = d[:i]
open(p, 'wb').write(d)
EOF
$T/../venv/bin/python $T/render.py $raw $cols $rows >| $T/$n-$state-$cols.html
grep -a -oE '(v[0-9]+\.zsh|core\.zsh|[a-z_]+):[0-9]+: [^\r]*' $raw | sort -u | head -5
