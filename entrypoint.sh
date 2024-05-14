#!/bin/bash

cleanup() {
    echo "SIGTERM received, shutting down..."
    exit 0
}

trap 'cleanup' SIGTERM

port=1936
http_response="HTTP/1.1 200 OK\r\nConnection: close\r\nContent-Type: text/plain\r\n\r\nHealthy!"

while true; do
    socat TCP-LISTEN:1936,crlf,reuseaddr SYSTEM:"echo \$(date) - Received request from \$SOCAT_PEERADDR; printf '%b' \"$http_response\""
    sleep 0.5
done
