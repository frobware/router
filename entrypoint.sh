#!/bin/bash

cleanup() {
    echo "SIGTERM received, shutting down..."
    exit 0
}

trap 'cleanup' SIGTERM

dropbear -E -p 2222

/usr/bin/entrypoint

