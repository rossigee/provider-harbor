FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --chmod=0755 bin/linux_amd64/provider /usr/local/bin/provider
COPY --chmod=0755 package/ /package/

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/provider"]