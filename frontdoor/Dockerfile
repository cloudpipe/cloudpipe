FROM golang:1.4

RUN useradd rho && \
  go get github.com/tools/godep && \
  chown -R rho:rho /go

USER rho

ADD ./Godeps /go/src/github.com/smashwilson/rhocloud/frontdoor/Godeps
WORKDIR /go/src/github.com/smashwilson/rhocloud/frontdoor/
RUN godep restore

ADD . /go/src/github.com/smashwilson/rhocloud/frontdoor/
RUN go install github.com/smashwilson/rhocloud/frontdoor

CMD ["/go/bin/frontdoor"]
