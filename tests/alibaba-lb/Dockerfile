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
	go mod tidy && mkdir -p /out/ms-25806 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-25806/main ./services/MS_25806 && \
	mkdir -p /out/ms-2687 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-2687/main ./services/MS_2687 && \
	mkdir -p /out/ms-40087 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-40087/main ./services/MS_40087 && \
	mkdir -p /out/ms-44246 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-44246/main ./services/MS_44246 && \
	mkdir -p /out/ms-51787 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-51787/main ./services/MS_51787 && \
	mkdir -p /out/ms-56113 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-56113/main ./services/MS_56113 && \
	mkdir -p /out/ms-64512 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-64512/main ./services/MS_64512 && \
	mkdir -p /out/ms-70124 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-70124/main ./services/MS_70124 && \
	mkdir -p /out/ms-64512-grpc && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-64512-grpc/main ./services/ms-64512-grpc && \
	mkdir -p /out/rajomon-client && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/rajomon-client/main ./services/rajomon-client

FROM alpine:latest AS svc-ms-25806
WORKDIR /root/
COPY --from=builder /out/ms-25806/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-2687
WORKDIR /root/
COPY --from=builder /out/ms-2687/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-40087
WORKDIR /root/
COPY --from=builder /out/ms-40087/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-44246
WORKDIR /root/
COPY --from=builder /out/ms-44246/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-51787
WORKDIR /root/
COPY --from=builder /out/ms-51787/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-56113
WORKDIR /root/
COPY --from=builder /out/ms-56113/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-64512
WORKDIR /root/
COPY --from=builder /out/ms-64512/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-70124
WORKDIR /root/
COPY --from=builder /out/ms-70124/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-64512-grpc
WORKDIR /root/
COPY --from=builder /out/ms-64512-grpc/main .
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

