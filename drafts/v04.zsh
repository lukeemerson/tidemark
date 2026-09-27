#!/usr/bin/env zsh
# 04 · instrument panel — one outer frame, sections split by shared ├┼┤ dividers
source ${0:A:h}/core.zsh
FRAME=round

seg() { # title, right label, width -> REPLY a divider segment spanning width
	local t=" $1 " rt=${2:+ $2 } n
	vis "$t$rt"
	n=$(($3 - 2 - REPLY))
	((n < 0)) && n=0
	rep ─ $n
	REPLY="$FR─$R$TTL$t$R$FR$REPLY$R$rt$FR─$R"
}

band() { # junctions (left mid right), left title, left label, right title, right label; L and R arrays -> out
	local j=($=1) i n
	seg "$2" "$3" $lw; local s1=$REPLY
	seg "$4" "$5" $rw
	out+=("$FR$j[1]$R$s1$FR$j[2]$R$REPLY$FR$j[3]$R")
	n=$(($#L > $#Rt ? $#L : $#Rt))
	for ((i = 1; i <= n; i++)); do
		padv "$L[i]" $((lw - 2)); local l=$REPLY
		padv "$Rt[i]" $((rw - 2))
		out+=("$FR│$R $l $FR│$R $REPLY $FR│$R")
	done
}

layout() {
	local rows=$1 w=$2 lw rw ph a b i
	local -a L Rt
	level $tc; a=$REPLY; num '%.0f°' $tc; a+="$REPLY$R"
	level $tg; b=$REPLY; num '%.0f°' $tg; b+="$REPLY$R"
	p_head $((w - 2))
	out=(" $REPLY")
	lw=$(((w - 3) * 3 / 5)) rw=$((w - 3 - lw))

	p_cpu $((lw - 2)) 6; L=("${reply[@]}")
	p_cores $((rw - 2)) 2; Rt=("${reply[@]}" "" "$DIM cpu $R$a$DIM  gpu $R$b$DIM  ·  $thermal  ·  ${fan%.*} rpm$R")
	h_cpu; local c=$REPLY
	band "╭ ┬ ╮" cpu "$c" cores ""

	p_gpu $((lw - 2)) 4; L=("${reply[@]}")
	p_pow $((rw - 2)) 4; Rt=("${reply[@]}")
	h_gpu; a=$REPLY; h_pow
	band "├ ┼ ┤" gpu "$a" power "$REPLY"

	p_mem $((lw - 2)); L=("${reply[@]}" "")
	p_io $((lw - 2)); L+=("${reply[@]}")
	p_cf $((rw - 2)); Rt=("${reply[@]}")
	h_mem; a=$REPLY; cf_age
	band "├ ┼ ┤" memory "$a" cloudflare "$DIM$REPLY$R"

	# full-width processes band, then the bottom edge
	seg processes "" $lw; b=$REPLY
	rep ─ $rw
	out+=("$FR├$R$b$FR┴$REPLY┤$R")
	ph=$((rows - $#out - 1))
	((ph < 3)) && ph=3
	p_proc $((w - 4)) $((ph - 1))
	for ((i = 1; i <= $#reply; i++)); do
		padv "$reply[i]" $((w - 4))
		out+=("$FR│$R $REPLY $FR│$R")
	done
	rep ─ $((w - 2))
	out+=("$FR╰$REPLY╯$R")
}

run
