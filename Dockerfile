FROM golang:1.18 AS exporter

ENV GOBIN=/go/bin
ENV GOPATH=/go
ENV CGO_ENABLED=0
ENV GOOS=linux

RUN git clone "https://github.com/validance/cosmos-exporter" /exporter
WORKDIR /exporter
RUN go install

FROM debian:buster-slim

RUN apt-get update && apt-get upgrade && apt-get install -y curl
RUN useradd -ms /bin/bash exporter && chown -R exporter /usr

COPY --from=exporter /go/bin/main /usr/bin/cosmos-exporter

USER exporter
