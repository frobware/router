FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base-router
RUN INSTALL_PKGS="haproxy22 rsyslog sysvinit-tools" && \
    yum install -y $INSTALL_PKGS && \
    rpm -V $INSTALL_PKGS && \
    yum clean all && \
    mkdir -p /tmp/lib/haproxy/router/{certs,cacerts,whitelists} && \
    mkdir -p /tmp/lib/haproxy/{conf/.tmp,run,bin,log} && \
    touch /tmp/lib/haproxy/conf/{{os_http_be,os_edge_reencrypt_be,os_tcp_be,os_sni_passthrough,os_route_http_redirect,cert_config,os_wildcard_domain}.map,haproxy.config} && \
    setcap 'cap_net_bind_service=ep' /usr/sbin/haproxy && \
    chown -R :0 /tmp/lib/haproxy && \
    chmod -R g+w /tmp/lib/haproxy
COPY images/router/haproxy/ /tmp/lib/haproxy/
LABEL io.k8s.display-name="OpenShift HAProxy Router" \
      io.k8s.description="This component offers ingress to an OpenShift cluster via Ingress and Route rules." \
      io.openshift.tags="openshift,router,haproxy"
USER 1001
EXPOSE 80 443
WORKDIR /tmp/lib/haproxy/conf
ENV TEMPLATE_FILE=/tmp/lib/haproxy/conf/haproxy-config.template \
    RELOAD_SCRIPT=/tmp/lib/haproxy/reload-haproxy
ENTRYPOINT ["/usr/bin/openshift-router", "--v=2"]
