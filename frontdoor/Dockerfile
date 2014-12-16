FROM golang:1.4

ADD . /go/src/github.com/smashwilson/rhocloud/frontdoor/

RUN useradd rho && \
  chown -R rho:rho /go

USER rho

WORKDIR /go/src/github.com/smashwilson/rhocloud/frontdoor/
RUN go get github.com/smashwilson/rhocloud/frontdoor

CMD ["/go/bin/frontdoor"]
