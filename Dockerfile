FROM golang:1.4.1

ENV GOPATH /
ENV PORT 8000
ENV ADMIN_PORT 8001

RUN mkdir -p /src/github.com/dynport/drp
EXPOSE 8000
EXPOSE 8001

ADD . /src/github.com/dynport/drp

RUN go get -v github.com/dynport/drp

ENTRYPOINT ["/bin/drp"]
