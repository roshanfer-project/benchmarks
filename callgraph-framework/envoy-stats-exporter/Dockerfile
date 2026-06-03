FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o envoy-stats-exporter .

FROM alpine:3.20
COPY --from=builder /build/envoy-stats-exporter /envoy-stats-exporter
ENTRYPOINT ["/envoy-stats-exporter"]
