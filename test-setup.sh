#!/usr/bin/env bash

rm -rf /tmp/var/lib/haproxy
mkdir -p /tmp/var/lib/haproxy/router/{certs,cacerts,whitelists}
mkdir -p /tmp/var/lib/haproxy/{conf/.tmp,run,bin,log}
touch /tmp/var/lib/haproxy/conf/{{os_http_be,os_edge_reencrypt_be,os_tcp_be,os_sni_passthrough,os_route_http_redirect,cert_config,os_wildcard_domain}.map,haproxy.config}
cp -a images/router/haproxy/* /tmp/var/lib/haproxy
