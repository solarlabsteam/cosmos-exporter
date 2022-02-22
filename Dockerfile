FROM golang:1.16-alpine as builder

RUN apk update && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

RUN adduser -D -g '' appuser

WORKDIR /app

ENV CGO_ENABLED=0

COPY go.mod .
COPY go.sum .
RUN go mod download
RUN go mod verify

COPY . .
RUN go build -o /go/bin/cosmos-exporter -ldflags '-extldflags "-static"'

# second step to build minimal image
FROM scratch

# add common trusted certificates from the build stage
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

USER appuser

COPY --from=builder /go/bin/cosmos-exporter /go/bin/cosmos-exporter

ENTRYPOINT ["/go/bin/cosmos-exporter"]
