# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS deps
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

FROM deps AS builder
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod tidy && mkdir -p /out/backend1 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/backend1/main ./services/backend1 && \
	mkdir -p /out/backend2 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/backend2/main ./services/backend2 && \
	mkdir -p /out/frontend && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/frontend/main ./services/frontend && \
	mkdir -p /out/shared && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/shared/main ./services/shared && \
	mkdir -p /out/frontend-grpc && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/frontend-grpc/main ./services/frontend-grpc && \
	mkdir -p /out/rajomon-client && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/rajomon-client/main ./services/rajomon-client

FROM alpine:latest AS svc-backend1
WORKDIR /root/
COPY --from=builder /out/backend1/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-backend2
WORKDIR /root/
COPY --from=builder /out/backend2/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-frontend
WORKDIR /root/
COPY --from=builder /out/frontend/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-shared
WORKDIR /root/
COPY --from=builder /out/shared/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-frontend-grpc
WORKDIR /root/
COPY --from=builder /out/frontend-grpc/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-rajomon-client
WORKDIR /root/
COPY --from=builder /out/rajomon-client/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

