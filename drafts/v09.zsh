#!/usr/bin/env zsh
# 09 · compact — sized for a zellij pane (~90×26); dashed frames, short graphs
source ${0:A:h}/core.zsh
FRAME=dashed

sys() { # inner width -> one line each: mem, power, temps, io
	local iw=$1 a b
	h_mem; a=$REPLY
	bar $((mt ? mu * 100.0 / mt : 0)) $((iw - 12))
	reply=("$DIM mem   $R$a $REPLY")
	h_pow; a=$REPLY; num '%.1f' $pwg
	reply+=("$DIM power $R$a$DIM  gpu $REPLY W$R")
	level $tc; a=$REPLY; num '%.0f°' $tc; a+="$REPLY$R"
	level $tg; b=$REPLY; num '%.0f°' $tg; b+="$REPLY$R"
	reply+=("$DIM temp  $R$a$DIM cpu  $R$b$DIM gpu  · ${thermal:-—}$R")
	rate $nin; a=$REPLY; rate $nout
	reply+=("$DIM net   $R$TEXT↓ $a ↑ $REPLY$R")
}

cf_small() { # inner width
	local iw=$1 a b
	if ((!cfn)); then reply=("$DIM no saved runs$R"); return; fi
	printf -v a '%.0f' $cfdl; printf -v b '%.0f' $cful
	reply=("$NET$TITLE↓ $a$R $POWER$TITLE↑ $b$R$DIM Mbps$R")
	printf -v a '%.0f' $cflat; printf -v b '%.0f' $cfjit
	reply+=("$DIM ping $R$TEXT$a$R$DIM  jit $R$TEXT$b$R$DIM ms  $cfbb/$cfstab$R")
	hmax cfhdl 1; spark cfhdl $((cfn < iw ? cfn : iw)) $REPLY; reply+=("$DIM runs $R$NET$REPLY$R")
	cf_age; reply+=("$DIM $cfcolo · $REPLY$R")
}

layout() {
	local rows=$1 w=$2 cw ph
	local -a a b
	p_head $w
	out=("$REPLY")
	cw=$(((w - 1) / 2))
	h_cpu; panel cpu "$REPLY" $cw 6 p_cpu 3; a=("${reply[@]}")
	h_gpu; panel gpu "$REPLY" $((w - cw - 1)) 6 p_gpu 3; b=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")
	sys $((cw - 4)); box system "" $cw 6 "${reply[@]}"; a=("${reply[@]}")
	cf_small $((w - cw - 5)); box cloudflare "" $((w - cw - 1)) 6 "${reply[@]}"; b=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")
	ph=$((rows - $#out))
	((ph < 4)) && ph=4
	panel processes "" $w $ph p_proc $((ph - 3)); out+=("${reply[@]}")
}

run
