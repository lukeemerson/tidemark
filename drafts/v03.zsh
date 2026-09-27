#!/usr/bin/env zsh
# 03 · tiles — a row of heavy KPI tiles (value + sparkline), graphs under, processes last
source ${0:A:h}/core.zsh
FRAME=heavy

tile() { # title, value, spark history, spark max, spark colour, width -> reply
	local tw=$6 iw=$(($6 - 4))
	center "$2" $iw; local v=$REPLY
	if [[ -n $3 ]]; then spark $3 $iw $4; REPLY="$5$REPLY$R"; else REPLY=; fi
	box "$1" "" $tw 4 "$v" "$REPLY"
}

layout() {
	local rows=$1 w=$2 tw last cw ph m v
	local -a t a b c
	p_head $w
	out=("$REPLY")
	tw=$(((w - 7) / 8)) last=$((w - 7 * tw - 7))
	h_cpu; tile cpu "$REPLY" hcpu 100 "" $tw; t=("${reply[@]}")
	h_gpu; tile gpu "$REPLY" hgpu 100 $GPU $tw; a=("${reply[@]}"); hjoin 1 t a; t=("${reply[@]}")
	hmax hpow 1; m=$REPLY
	h_pow; tile power "$REPLY" hpow $m $POWER $tw; a=("${reply[@]}"); hjoin 1 t a; t=("${reply[@]}")
	h_mem; tile mem "$REPLY" hmem 100 $MID $tw; a=("${reply[@]}"); hjoin 1 t a; t=("${reply[@]}")
	level $tc; v=$REPLY; num '%.0f°' $tc; ((have)) && REPLY="$TITLE$v$REPLY$R"
	tile temp "$REPLY" htc 110 $HIGH $tw; a=("${reply[@]}"); hjoin 1 t a; t=("${reply[@]}")
	hmax cfhdl 1; m=$REPLY; printf -v v '%.0f Mbps' $cfdl
	tile "↓ cf" "$NET$TITLE$v$R" cfhdl $m $NET $tw; a=("${reply[@]}"); hjoin 1 t a; t=("${reply[@]}")
	hmax cfhul 1; m=$REPLY; printf -v v '%.0f Mbps' $cful
	tile "↑ cf" "$POWER$TITLE$v$R" cfhul $m $POWER $tw; a=("${reply[@]}"); hjoin 1 t a; t=("${reply[@]}")
	hmax cfhlat 1; m=$REPLY; printf -v v '%.0f ms' $cflat
	tile ping "$TITLE$v$R" cfhlat $m $DIM $last; a=("${reply[@]}"); hjoin 1 t a
	out+=("${reply[@]}")

	cw=$(((w - 1) / 2))
	h_cpu; panel cpu "$REPLY" $cw 9 p_cpu 6; a=("${reply[@]}")
	h_gpu; panel gpu "$REPLY" $((w - cw - 1)) 9 p_gpu 6; b=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")
	panel cores "" $cw 7 p_cores 2; a=("${reply[@]}")
	h_pow; panel power "$REPLY" $((w - cw - 1)) 7 p_pow 4; b=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")

	ph=$((rows - $#out))
	((ph < 8)) && ph=8
	cw=$((w * 2 / 3))
	panel processes "" $cw $ph p_proc $((ph - 3)); a=("${reply[@]}")
	h_mem; panel memory "$REPLY" $((w - cw - 1)) 5 p_mem; b=("${reply[@]}")
	panel sensors "" $((w - cw - 1)) 5 p_sens; b+=("${reply[@]}")
	panel io "" $((w - cw - 1)) 5 p_io; b+=("${reply[@]}")
	cf_age; panel cloudflare "$DIM$REPLY$R" $((w - cw - 1)) $((ph - 15)) p_cf; b+=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")
}

run
