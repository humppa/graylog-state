# vim: set syntax=dockerfile :

FROM golang:alpine

RUN set -x \
  && apk add --no-cache bash curl git \
  && adduser -u 1000 -G users -D go

USER go

WORKDIR /go

ENTRYPOINT ["/bin/bash"]
