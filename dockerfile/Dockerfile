FROM golang:alpine AS cosmos-exporter
RUN apk add --no-cache git make gcc libc-dev
WORKDIR /go/src/github.com/DefiantLabs
RUN git clone https://github.com/DefiantLabs/cosmos-val-exporter.git
WORKDIR /go/src/github.com/DefiantLabs/cosmos-val-exporter
ARG VERSION=v0.3.2
RUN git checkout ${VERSION}
RUN COMMIT="$(git rev-parse HEAD)"; go install -mod=readonly -tags "muslc" -ldflags "-X cosmos-exporter/cmd.Version=${VERSION}-$COMMIT -X cosmos-exporter/cmd.Commit=$COMMIT" ./...
FROM alpine:latest 
COPY --from=cosmos-exporter /go/bin/main /usr/local/bin/cosmos-exporter
RUN apk add --no-cache gcc
COPY commands.sh /scripts/commands.sh
RUN ["chmod", "+x", "/scripts/commands.sh"]
ENTRYPOINT ["/scripts/commands.sh"]