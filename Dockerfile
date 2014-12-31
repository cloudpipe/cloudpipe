FROM golang:1.4

RUN useradd rho && \
  go get github.com/tools/godep && \
  chown -R rho:rho /go

# USER rho

ADD ./Godeps /go/src/github.com/cloudpipe/cloudpipe/frontdoor/Godeps
WORKDIR /go/src/github.com/cloudpipe/cloudpipe/frontdoor/
RUN godep restore

ADD . /go/src/github.com/cloudpipe/cloudpipe/frontdoor/
RUN go install github.com/cloudpipe/cloudpipe/frontdoor

CMD ["/go/bin/frontdoor"]
