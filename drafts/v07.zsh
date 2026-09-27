#!/usr/bin/env zsh
# 07 · three columns — compute | graphics & power | memory & network, double frames
source ${0:A:h}/core.zsh
FRAME=double

layout() {
	local rows=$1 w=$2 cw c3 ph
	local -a a b c
	p_head $w
	out=("$REPLY")
	cw=$(((w - 2) / 3)) c3=$((w - 2 * cw - 2))
	h_cpu; panel cpu "$REPLY" $cw 9 p_cpu 6; a=("${reply[@]}")
	panel cores "" $cw 7 p_cores 2; a+=("${reply[@]}")
	panel sensors "" $cw 5 p_sens; a+=("${reply[@]}")
	h_gpu; panel gpu "$REPLY" $cw 8 p_gpu 5; b=("${reply[@]}")
	h_pow; panel power "$REPLY" $cw 8 p_pow 5; b+=("${reply[@]}")
	h_mem; panel memory "$REPLY" $cw 5 p_mem; b+=("${reply[@]}")
	panel io "" $c3 5 p_io; c=("${reply[@]}")
	cf_age; panel cloudflare "$DIM$REPLY$R" $c3 16 p_cfbars 6; c+=("${reply[@]}")
	hjoin 1 a b c; out+=("${reply[@]}")

	ph=$((rows - $#out))
	((ph < 4)) && ph=4
	panel processes "" $w $ph p_proc $((ph - 3)); out+=("${reply[@]}")
}

run
