#!/usr/bin/env bash

cleanup() {
    echo "Signal received, shutting down..."
    pkill openshift-router
    pkill openshift-router
    pkill openshift-router
    sleep 1
    pgrep openshift-router
    pkill dlv
    exit 0
}

trap 'cleanup' SIGTERM SIGINT

dlv exec --log --api-version=2 --headless --accept-multiclient --continue --listen=:7000 /usr/bin/openshift-router -- --v=2 2>&1 | tee /proc/1/fd/1

