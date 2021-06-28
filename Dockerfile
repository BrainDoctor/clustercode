FROM docker.io/library/alpine:3.14 as runtime

ENTRYPOINT ["clustercode"]

RUN \
    apk add --no-cache curl bash

COPY clustercode /usr/bin/
USER 1001:0
