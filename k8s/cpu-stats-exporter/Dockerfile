# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY go.mod main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o cpu-stats-exporter .

# Final stage
FROM scratch
COPY --from=builder /build/cpu-stats-exporter /cpu-stats-exporter
ENTRYPOINT ["/cpu-stats-exporter"]
