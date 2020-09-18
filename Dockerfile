FROM quay.io/openshift/origin-cli:4.5 as builder

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

RUN microdnf update -y \
    && microdnf install -y tar rsync findutils gzip iproute util-linux \
    && microdnf clean all

# Copy oc binary
COPY --from=builder /usr/bin/oc /usr/bin/oc

COPY test/* /opt/

ENTRYPOINT ["/opt/e2etest.sh"]

