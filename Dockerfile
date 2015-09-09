FROM jbenet/go-ipfs

ENV IPFS_LOGGING info

USER root

ENV DIST_DIR /go/src/github.com/docker/distribution
ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH
ENV GOPATH $DIST_DIR/Godeps/_workspace:$GOPATH

WORKDIR $DIST_DIR
COPY . $DIST_DIR
COPY demo/config-ipfs.yml /etc/docker/registry/config.yml
COPY demo/start_registry /usr/local/bin/start_registry
COPY demo/supervisord.conf /etc/supervisord.conf

RUN apk add apache2-utils make go=1.4.2-r0 git
RUN make PREFIX=/go clean binaries
RUN apk del apache2-utils make git

VOLUME ["/var/lib/registry"]
EXPOSE 5000

RUN apk add supervisor
RUN mkdir -p /supervisord/log
RUN chown -R ipfs:ipfs /supervisord
WORKDIR /supervisord

ENTRYPOINT ["/usr/bin/supervisord"]
CMD ["-c", "/etc/supervisord.conf"]
