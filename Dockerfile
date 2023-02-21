FROM golang:1.19-alpine AS builder

COPY . /app

WORKDIR /app

RUN go build -o cosmos-exporter


FROM alpine

COPY --from=builder /app/cosmos-exporter /usr/local/bin/cosmos-exporter

ENTRYPOINT [ "/usr/local/bin/cosmos-exporter" ]
