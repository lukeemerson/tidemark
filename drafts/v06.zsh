#!/usr/bin/env zsh
# 06 · sidebar — every number in one narrow column; the rest of the screen is graphs
source ${0:A:h}/core.zsh
FRAME=round

row() { # label, value, [bar percent] -> line in the sidebar
	local l="$DIM${(r:7:)1}$R$2"
	if [[ -n $3 ]]; then
		bar $3 $((iw - 21)) "" ▰ ▱
		local b=$REPLY
		vis $l; l="${l}${(l:$((iw - REPLY - (iw - 21))):: :)${:-}}$b"
	fi
	REPLY=$l
}

side() { # inner width, height -> reply
	local iw=$1 h=$2 a b i m c
	local -a s
	h_cpu; row cpu "$REPLY" $cpu; s+=("$REPLY")
	c=
	for ((i = 1; i <= ne + np; i++)); do
		level ${cores[i]:-0}; c+=$REPLY
		m=$((int(${cores[i]:-0} / 100.0 * 7 + 0.5)))
		c+=$SPK[m+1]
	done
	row cores "$c$R"; s+=("$REPLY")
	h_gpu; row gpu "$REPLY" $gpu; s+=("$REPLY")
	h_mem; row mem "$REPLY" $((mt ? mu * 100.0 / mt : 0)); s+=("$REPLY")
	h_pow; row power "$REPLY"; s+=("$REPLY" "")
	level $tc; a=$REPLY; num '%.0f°' $tc; a+="$REPLY$R"
	level $tg; b=$REPLY; num '%.0f°' $tg; b+="$REPLY$R"
	row temp "$a$DIM cpu  $R$b$DIM gpu$R"; s+=("$REPLY")
	num '%.0f' $fan; row fan "$TEXT$REPLY$R$DIM rpm · ${thermal:-—}$R"; s+=("$REPLY")
	rate $nin; a=$REPLY; rate $nout
	row net "$TEXT↓ $a ↑ $REPLY$R"; s+=("$REPLY")
	rate $((dkr * 1024)); a=$REPLY; rate $((dkw * 1024))
	row disk "${TEXT}r $a w $REPLY$R"; s+=("$REPLY")
	if ((vpct >= 0)); then row "${vname[1,6]}" "$TEXT$(printf '%.0f%%' $vpct)$R" $vpct; s+=("$REPLY"); fi
	s+=("")
	rep ─ $((iw - 14)); s+=("$FR── cloudflare $REPLY$R")
	if ((cfn)); then
		printf -v a '%.0f' $cfdl; printf -v b '%.0f' $cful
		row speed "$NET$TITLE↓ $a$R$DIM / $R$POWER$TITLE↑ $b$R$DIM Mbps$R"; s+=("$REPLY")
		printf -v a '%.0f' $cflat; printf -v b '%.0f' $cfjit
		row ping "$TEXT$a ms$R$DIM  jitter $R$TEXT$b ms$R"; s+=("$REPLY")
		row grade "$DIM bloat $R$TEXT$cfbb$R$DIM  stable $R$TEXT$cfstab$R"; s+=("$REPLY")
		cf_age; row last "$DIM$REPLY · $cfcolo · $cfif${${cfwifi:#0}:+ wi-fi}$R"; s+=("$REPLY")
		# leftover height: one ↓/↑ bar pair per saved run
		local ch=$((h - $#s - 3))
		((ch > 10)) && ch=10
		if ((ch >= 3)); then
			cf_tail $(((iw - 5) / 3))
			hmax cfhdl 1; m=$REPLY
			vbar2 CFD CFU $ch $m 1 $NET $POWER
			s+=("")
			for ((i = 1; i <= $#reply; i++)); do
				((i == 1)) && printf -v a '%4.0f ' $m || a="     "
				s+=("$DIM$a$R$reply[i]")
			done
			s+=("$DIM     $cfn runs · $R$NET▌$R$DIM down $R$POWER▌$R$DIM up$R")
		fi
	else
		s+=("$DIM no saved runs — run: cloudy$R")
	fi
	reply=("${s[@]}")
}

layout() {
	local rows=$1 w=$2 sw=38 gw ph
	local -a a b
	p_head $w
	out=("$REPLY")
	gw=$((w - sw - 1))
	side $((sw - 4)) $((rows - 3)); box "$name" "" $sw $((rows - 1)) "${reply[@]}"; a=("${reply[@]}")
	h_cpu; panel "cpu" "$DIM$(printf 'E %.0f · P %.0f MHz' $efreq $pfreq)$R  $REPLY" $gw 9 p_cpu 6; b=("${reply[@]:0:1}" "${reply[@]:2}")
	h_gpu; panel gpu "$REPLY" $gw 7 p_gpu 4; b+=("${reply[@]}")
	h_pow; panel power "$REPLY" $gw 7 p_pow 4; b+=("${reply[@]}")
	ph=$((rows - 1 - $#b))
	((ph < 4)) && ph=4
	panel processes "" $gw $ph p_proc $((ph - 3)); b+=("${reply[@]}")
	hjoin 1 a b; out+=("${reply[@]}")
}

run
