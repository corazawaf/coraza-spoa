#!/bin/sh

set -e

# Allow users to run arbitrary commands within the container

if [ $# -gt 0 ] && [ "$1" = "${1#-}" ]; then
    # First char isn't `-`, probably a `docker run -ti <cmd>`
    # Just exec and exit
    exec "$@"
    exit
fi

exec coraza-spoa --config /etc/coraza-spoa/config.yaml
