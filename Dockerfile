# first image used to build the sources
FROM golang:1.18-buster AS builder

ARG VERSION
ARG COMMIT
ARG DATE
ARG TARGETOS
ARG TARGETARCH


ENV GO111MODULE=on \
    CGO_ENABLED=1

WORKDIR /neutrino-elements

COPY . .

RUN go mod download

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT}' -X 'main.date=${DATE}'" -o ./bin/neutrinod cmd/neutrinod/main.go
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -X 'main.version=${VERSION}' -X 'main.commit=${COMMIT}' -X 'main.date=${DATE}'" -o ./bin/neutrino cmd/neutrino/*

# Second image, running the towerd executable
FROM debian:buster-slim
ENV NEUTRINO_ELEMENTS_DB_MIGRATION_PATH="file://"

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates

WORKDIR /app

COPY --from=builder /neutrino-elements/bin/neutrinod .
COPY --from=builder /neutrino-elements/bin/neutrino .
COPY --from=builder /neutrino-elements/internal/infrastructure/storage/db/pg/migration/* .

RUN install neutrino /bin
RUN install neutrinod /bin

ENTRYPOINT ["./neutrinod"]
