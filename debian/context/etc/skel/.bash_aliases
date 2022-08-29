resize() {
	if [[ -t 0 && $# -eq 0 ]]; then
		local IFS='[;' escape geometry x y
		echo -ne '\e7\e[r\e[999;999H\e[6n\e8'
		read -sd R escape geometry
		x=${geometry##*;} y=${geometry%%;*}
		if [[ ${COLUMNS} -eq ${x} && ${LINES} -eq ${y} ]]; then
			echo "${TERM} ${x}x${y}"
		else
			echo "${COLUMNS}x${LINES} -> ${x}x${y}"
			stty cols ${x} rows ${y}
		fi
	else
		print 'Usage: resize'
	fi
}

case $(/usr/bin/tty) in
/dev/ttyS1)
    export LANG=C
	resize
	;;
esac
