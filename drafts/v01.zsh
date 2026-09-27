#!/usr/bin/env zsh
# 01 · btop classic — rounded boxes, CPU across the top, processes take the slack
source ${0:A:h}/core.zsh
FRAME=round

layout() {
	local rows=$1 w=$2 lw rw tw ph
	local -a a b c
	p_head $w
	out=("$REPLY")
	lw=$((w * 3 / 5)) rw=$((w - lw - 1))
	h_cpu; panel cpu "$REPLY" $lw 10 p_cpu 7; a=("${reply[@]}")
	panel cores "" $rw 10 p_cores 2; b=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")

	tw=$(((w - 2) / 3))
	h_mem; panel mem "$REPLY" $tw 7 p_mem; a=("${reply[@]}")
	h_gpu; panel gpu "$REPLY" $tw 7 p_gpu 4; b=("${reply[@]}")
	h_pow; panel power "$REPLY" $((w - 2 * tw - 2)) 7 p_pow 4; c=("${reply[@]}")
	hjoin 1 a b c; out+=("${reply[@]}")

	ph=$((rows - $#out))
	((ph < 12)) && ph=12
	panel sensors "" $rw 5 p_sens; a=("${reply[@]}")
	panel io "" $rw 5 p_io; a+=("${reply[@]}")
	cf_age; panel cloudflare "$DIM$REPLY$R" $rw $((ph - 10)) p_cf; a+=("${reply[@]}")
	panel processes "" $lw $ph p_proc $((ph - 3)); b=("${reply[@]}")
	hjoin 1 b a; out+=("${reply[@]}")
}

run
