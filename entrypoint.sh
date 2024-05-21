#!/bin/bash

cleanup() {
    echo "SIGTERM received, shutting down..."
    pkill -x -f /usr/sbin/sshd
    exit 0
}

trap 'cleanup' SIGTERM SIGINT

/usr/sbin/sshd
/usr/bin/entrypoint &
wait $!

