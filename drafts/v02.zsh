#!/usr/bin/env zsh
# 02 · two stacks — the current layout, boxed: CPU side | SoC side, processes full width
source ${0:A:h}/core.zsh
FRAME=square

layout() {
	local rows=$1 w=$2 cw ph
	local -a a b
	p_head $w
	out=("$REPLY")
	cw=$(((w - 1) / 2))
	h_cpu; panel cpu "$REPLY" $cw 9 p_cpu 6; a=("${reply[@]}")
	panel cores "" $cw 7 p_cores 2; a+=("${reply[@]}")
	h_mem; panel memory "$REPLY" $cw 6 p_mem; a+=("${reply[@]}")
	h_gpu; panel gpu "$REPLY" $((w - cw - 1)) 7 p_gpu 4; b=("${reply[@]}")
	h_pow; panel power "$REPLY" $((w - cw - 1)) 6 p_pow 3; b+=("${reply[@]}")
	cf_age; panel cloudflare "$DIM$REPLY$R" $((w - cw - 1)) 9 p_cf; b+=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")

	panel sensors "" $cw 5 p_sens; a=("${reply[@]}")
	panel io "" $((w - cw - 1)) 5 p_io; b=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")

	ph=$((rows - $#out))
	((ph < 5)) && ph=5
	panel processes "" $w $ph p_proc $((ph - 3)); out+=("${reply[@]}")
}

run
