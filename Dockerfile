FROM golang:1.4

RUN apt-get update && \
    apt-get install -y librados-dev apache2-utils && \
    rm -rf /var/lib/apt/lists/*

ENV DISTRIBUTION_DIR /go/src/github.com/docker/distribution
ENV GOPATH $DISTRIBUTION_DIR/Godeps/_workspace:$GOPATH
ENV DOCKER_BUILDTAGS include_rados include_oss

WORKDIR $DISTRIBUTION_DIR
COPY . $DISTRIBUTION_DIR
COPY cmd/registry/config-ipfs.yml /etc/docker/registry/config.yml
RUN make PREFIX=/go clean binaries
# Wait for the ipfs API to start
RUN echo '#!/bin/bash\necho waiting 10 secs...\nsleep 10 && registry "$@"' > /usr/bin/entrypoint.sh
RUN chmod +x /usr/bin/entrypoint.sh

VOLUME ["/var/lib/registry"]
EXPOSE 5000

ENTRYPOINT ["/usr/bin/entrypoint.sh"]
CMD ["/etc/docker/registry/config.yml"]
