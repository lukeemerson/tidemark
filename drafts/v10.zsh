#!/usr/bin/env zsh
# 10 · graph wall — full-width graphs stacked, their numbers live in the frame titles
source ${0:A:h}/core.zsh
FRAME=round

layout() {
	local rows=$1 w=$2 iw=$(($2 - 4)) a b m tw ph
	local -a x y z
	p_head $w
	out=("$REPLY")
	num '%.0f' $efreq; a=$REPLY; num '%.0f' $pfreq; b=$REPLY
	h_cpu; x=("$DIM E $a · P $b MHz$R   $REPLY")
	graph hcpu $iw 6 100; box cpu "$x[1]" $w 8 "${reply[@]}"; out+=("${reply[@]}")
	num '%.0f' $gfreq; a=$REPLY
	h_gpu; b=$REPLY
	graph hgpu $iw 4 100 $GPU; box gpu "$DIM$a MHz · ANE ${ane%.*}%$R   $b" $w 6 "${reply[@]}"; out+=("${reply[@]}")
	hmax hpow 1; m=$REPLY
	h_pow; b=$REPLY
	graph hpow $iw 4 $m $POWER; box power "$DIM peak $(printf '%.1f' $m) W$R   $b" $w 6 "${reply[@]}"; out+=("${reply[@]}")
	hmax hnin 1024; m=$REPLY
	rate $nin; a=$REPLY; rate $nout; b=$REPLY
	graph hnin $iw 3 $m $NET; box "network in" "$DIM↓ $a  ↑ $b$R" $w 5 "${reply[@]}"; out+=("${reply[@]}")

	tw=$(((w - 2) / 3))
	panel cores "" $tw 7 p_cores 2; x=("${reply[@]}")
	p_mem $((tw - 4)); y=("${reply[@]}"); p_sens $((tw - 4)); y+=("${reply[@]:0:2}")
	h_mem; box "memory · sensors" "$REPLY" $tw 7 "${y[@]}"; y=("${reply[@]}")
	cf_age; panel cloudflare "$DIM$REPLY$R" $((w - 2 * tw - 2)) 7 p_cfstats; z=("${reply[@]}")
	hjoin 1 x y z; out+=("${reply[@]}")

	ph=$((rows - $#out))
	if ((ph >= 5)); then panel processes "" $w $ph p_proc $((ph - 3)); out+=("${reply[@]}"); fi
}

run
