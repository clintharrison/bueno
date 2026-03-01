#!/bin/sh

show_eink_log() {
	msg="$1"
	echo >&2 "printing to screen: $msg"
	# lmao. clear the area we want to write, i guess?
	# should probably just use the logging in libkh5
	eips 0 67 "                                   "
	eips 0 67 "$msg"
}

stop() {
	pkill "kindle-keymap"
}

start_if_not_running() {
	if ! pgrep -f "kindle-keymap" >/dev/null; then
		echo >&2 "starting kindle-keymap..."
	fi
	nohup /mnt/us/extensions/kindle-keymap/bin/kindle-keymap >>/tmp/kindle-keymap.log 2>&1 &
}

run_pairing() {
	KINDLE_KEYMAP_RUN_BLUETOOTH_PAIR=1 nohup /mnt/us/extensions/kindle-keymap/bin/kindle-keymap \
		>>/tmp/kindle-keymap.log 2>&1 &
}

case "$1" in
check-status)
	if pgrep "kindle-keymap" >/dev/null; then
		show_eink_log "kindle-keymap is running      "
	else
		show_eink_log "kindle-keymap is not running  "
	fi
	;;
start)
	show_eink_log "starting kindle-keymap...     "
	start_if_not_running
	;;
run-pair)
	run_pairing
	;;
stop)
	show_eink_log "stopping kindle-keymap...     "
	stop
	;;
esac
