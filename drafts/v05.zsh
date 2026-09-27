#!/usr/bin/env zsh
# 05 · rules — no boxes, just titled rules; the quietest frame
source ${0:A:h}/core.zsh
FRAME=rule

layout() {
	local rows=$1 w=$2 cw ph
	local -a a b
	p_head $w
	out=("$REPLY" "")
	cw=$(((w - 3) / 2))
	h_cpu; panel cpu "$REPLY" $cw 10 p_cpu 7; a=("${reply[@]}")
	panel cores "" $cw 7 p_cores 2; a+=("${reply[@]}")
	h_mem; panel memory "$REPLY" $cw 5 p_mem; a+=("${reply[@]}")
	panel io "" $cw 5 p_io; a+=("${reply[@]}")
	h_gpu; panel gpu "$REPLY" $((w - cw - 3)) 7 p_gpu 4; b=("${reply[@]}")
	h_pow; panel power "$REPLY" $((w - cw - 3)) 6 p_pow 3; b+=("${reply[@]}")
	panel sensors "" $((w - cw - 3)) 4 p_sens; b+=("${reply[@]}")
	cf_age; panel cloudflare "$DIM$REPLY$R" $((w - cw - 3)) 9 p_cf; b+=("${reply[@]}")
	hjoin 3 a b; out+=("${reply[@]}")

	ph=$((rows - $#out))
	((ph < 4)) && ph=4
	panel processes "" $w $ph p_proc $((ph - 2)); out+=("${reply[@]}")
}

run
