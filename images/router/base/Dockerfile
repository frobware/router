FROM registry.svc.ci.openshift.org/openshift/release:golang-1.12 AS builder
WORKDIR /go/src/github.com/openshift/router
COPY . .
RUN make

FROM registry.svc.ci.openshift.org/openshift/origin-v4.0:base
COPY --from=builder /go/src/github.com/openshift/router/openshift-router /usr/bin/
# TODO(frobware): This needs to be reproducible for now and for eternity 
RUN wget -O /usr/bin/catatonit https://github.com/openSUSE/catatonit/releases/download/v0.1.4/catatonit.x86_64 && chmod u+x /usr/bin/catatonit
LABEL io.k8s.display-name="OpenShift Router" \
      io.k8s.description="This is the base image from which all template based routers inherit." \
      io.openshift.tags="openshift,router"
