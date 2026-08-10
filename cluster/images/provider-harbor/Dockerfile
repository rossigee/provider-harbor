FROM debian:stable-slim

# OpenContainers Image Spec Labels
LABEL org.opencontainers.image.title="provider-harbor"
LABEL org.opencontainers.image.description="Crossplane provider for Harbor container registry"
LABEL org.opencontainers.image.vendor="rossigee"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.source="https://github.com/rossigee/provider-harbor"
LABEL org.opencontainers.image.url="https://github.com/rossigee/provider-harbor"
LABEL org.opencontainers.image.documentation="https://github.com/rossigee/provider-harbor/blob/master/README.md"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --chmod=0755 bin/linux_amd64/provider /usr/local/bin/provider
COPY --chmod=0755 package/ /package/

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/provider"]