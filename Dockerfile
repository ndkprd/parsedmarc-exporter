ARG GO_IMAGE=golang:1.27.1
ARG RUNTIME_IMAGE=debian:trixie-20260918-slim

FROM ${GO_IMAGE} AS builder
# Override to build behind a module mirror: --build-arg GOPROXY=https://proxy.example/go,direct
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/exporter ./cmd/exporter

FROM ${RUNTIME_IMAGE}
COPY --from=builder /out/exporter /usr/local/bin/exporter
USER nobody
EXPOSE 9190
ENTRYPOINT ["/usr/local/bin/exporter"]
