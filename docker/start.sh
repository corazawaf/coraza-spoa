#!/usr/bin/env sh
set -e

if [ $# -gt 0 ] && [ "$1" = "${1#-}" ]; then
	# First char isn't `-`, probably a `docker run -ti <cmd>`
	# Just exec and exit
	exec "$@"
	exit
fi

unset conf
while [ $# -gt 0 ]; do
	case "$1" in
	-f)
		shift
		conf="$1"
		;;
	esac
	shift
done

conf="${conf:-/config.yaml}"

if [ ! -f "$conf" ]; then
	echo "File not found: $conf" >&2
	exit 1
fi

echo "Using config file: $conf"

exec ./coraza-spoa -config $conf
