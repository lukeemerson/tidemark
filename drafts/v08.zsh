#!/usr/bin/env zsh
# 08 · network first — Cloudflare history leads, the machine sits in a compact band under it
source ${0:A:h}/core.zsh
FRAME=square

cf_big() { # inner width -> reply: stats on the left, run-by-run bars on the right
	local iw=$1 sw=52 m i l
	local -a L C
	p_cfstats $sw; L=("${reply[@]}" "")
	p_cfwhere; L+=("$REPLY" "")
	((cfn)) && L+=("$DIM ${cfn} saved runs  ·  $R$NET▌$R$DIM download  $R$POWER▌$R$DIM upload$R" "$DIM run $R${TEXT}cloudy$R$DIM for a fresh test$R")
	hmax cfhdl 1; m=$REPLY
	local n=$(((iw - sw - 8) / 5))
	cf_tail $n
	vbar2 CFD CFU 8 $m 2 $NET $POWER
	for ((i = 1; i <= $#reply; i++)); do
		((i == 1)) && printf -v l '%4.0f ' $m || l="     "
		C+=("$DIM$l$R$reply[i]")
	done
	hjoin 3 L C
}

layout() {
	local rows=$1 w=$2 qw last ph cw
	local -a a b c d
	p_head $w
	out=("$REPLY")
	cf_big $((w - 4)); cf_age
	box cloudflare "$DIM$REPLY$R" $w 10 "${reply[@]}"; out+=("${reply[@]}")

	qw=$(((w - 3) / 4)) last=$((w - 3 * qw - 3))
	h_cpu; panel cpu "$REPLY" $qw 7 p_cpu 4; a=("${reply[@]}")
	h_gpu; panel gpu "$REPLY" $qw 7 p_gpu 4; b=("${reply[@]}")
	h_pow; panel power "$REPLY" $qw 7 p_pow 4; c=("${reply[@]}")
	h_mem; panel memory "$REPLY" $last 7 p_mem; d=("${reply[@]}")
	hjoin 1 a b c d; out+=("${reply[@]}")

	ph=$((rows - $#out))
	((ph < 8)) && ph=8
	cw=$(((w - 1) * 2 / 3))
	panel processes "" $cw $ph p_proc $((ph - 3)); a=("${reply[@]}")
	panel cores "" $((w - cw - 1)) 7 p_cores 2; b=("${reply[@]}")
	panel sensors "" $((w - cw - 1)) 5 p_sens; b+=("${reply[@]}")
	panel io "" $((w - cw - 1)) $((ph - 12)) p_io; b+=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")
}

run
