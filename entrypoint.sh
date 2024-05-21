#!/bin/bash

cleanup() {
    echo "SIGTERM received, shutting down..."
    exit 0
}

trap 'cleanup' SIGTERM

/usr/sbin/sshd
/usr/bin/entrypoint

