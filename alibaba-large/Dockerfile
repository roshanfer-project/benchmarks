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
	go mod tidy && mkdir -p /out/ms-12657 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-12657/main ./services/MS_12657 && \
	mkdir -p /out/ms-14758 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-14758/main ./services/MS_14758 && \
	mkdir -p /out/ms-18750 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-18750/main ./services/MS_18750 && \
	mkdir -p /out/ms-19439 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-19439/main ./services/MS_19439 && \
	mkdir -p /out/ms-21298 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-21298/main ./services/MS_21298 && \
	mkdir -p /out/ms-25781 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-25781/main ./services/MS_25781 && \
	mkdir -p /out/ms-25806 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-25806/main ./services/MS_25806 && \
	mkdir -p /out/ms-2687 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-2687/main ./services/MS_2687 && \
	mkdir -p /out/ms-33572 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-33572/main ./services/MS_33572 && \
	mkdir -p /out/ms-38190 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-38190/main ./services/MS_38190 && \
	mkdir -p /out/ms-40087 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-40087/main ./services/MS_40087 && \
	mkdir -p /out/ms-41667 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-41667/main ./services/MS_41667 && \
	mkdir -p /out/ms-43032 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-43032/main ./services/MS_43032 && \
	mkdir -p /out/ms-43754 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-43754/main ./services/MS_43754 && \
	mkdir -p /out/ms-44246 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-44246/main ./services/MS_44246 && \
	mkdir -p /out/ms-45067 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-45067/main ./services/MS_45067 && \
	mkdir -p /out/ms-51783 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-51783/main ./services/MS_51783 && \
	mkdir -p /out/ms-51787 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-51787/main ./services/MS_51787 && \
	mkdir -p /out/ms-53792 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-53792/main ./services/MS_53792 && \
	mkdir -p /out/ms-56113 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-56113/main ./services/MS_56113 && \
	mkdir -p /out/ms-5720 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-5720/main ./services/MS_5720 && \
	mkdir -p /out/ms-58796 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-58796/main ./services/MS_58796 && \
	mkdir -p /out/ms-62039 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-62039/main ./services/MS_62039 && \
	mkdir -p /out/ms-64512 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-64512/main ./services/MS_64512 && \
	mkdir -p /out/ms-66921 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-66921/main ./services/MS_66921 && \
	mkdir -p /out/ms-67465 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-67465/main ./services/MS_67465 && \
	mkdir -p /out/ms-70124 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-70124/main ./services/MS_70124 && \
	mkdir -p /out/ms-7103 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-7103/main ./services/MS_7103 && \
	mkdir -p /out/ms-9105 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-9105/main ./services/MS_9105 && \
	mkdir -p /out/ms-64512-grpc && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/ms-64512-grpc/main ./services/ms-64512-grpc && \
	mkdir -p /out/rajomon-client && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/rajomon-client/main ./services/rajomon-client

FROM alpine:latest AS svc-ms-12657
WORKDIR /root/
COPY --from=builder /out/ms-12657/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-14758
WORKDIR /root/
COPY --from=builder /out/ms-14758/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-18750
WORKDIR /root/
COPY --from=builder /out/ms-18750/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-19439
WORKDIR /root/
COPY --from=builder /out/ms-19439/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-21298
WORKDIR /root/
COPY --from=builder /out/ms-21298/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-25781
WORKDIR /root/
COPY --from=builder /out/ms-25781/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

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

FROM alpine:latest AS svc-ms-33572
WORKDIR /root/
COPY --from=builder /out/ms-33572/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-38190
WORKDIR /root/
COPY --from=builder /out/ms-38190/main .
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

FROM alpine:latest AS svc-ms-41667
WORKDIR /root/
COPY --from=builder /out/ms-41667/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-43032
WORKDIR /root/
COPY --from=builder /out/ms-43032/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-43754
WORKDIR /root/
COPY --from=builder /out/ms-43754/main .
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

FROM alpine:latest AS svc-ms-45067
WORKDIR /root/
COPY --from=builder /out/ms-45067/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-51783
WORKDIR /root/
COPY --from=builder /out/ms-51783/main .
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

FROM alpine:latest AS svc-ms-53792
WORKDIR /root/
COPY --from=builder /out/ms-53792/main .
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

FROM alpine:latest AS svc-ms-5720
WORKDIR /root/
COPY --from=builder /out/ms-5720/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-58796
WORKDIR /root/
COPY --from=builder /out/ms-58796/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-62039
WORKDIR /root/
COPY --from=builder /out/ms-62039/main .
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

FROM alpine:latest AS svc-ms-66921
WORKDIR /root/
COPY --from=builder /out/ms-66921/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-67465
WORKDIR /root/
COPY --from=builder /out/ms-67465/main .
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

FROM alpine:latest AS svc-ms-7103
WORKDIR /root/
COPY --from=builder /out/ms-7103/main .
COPY --from=builder /app/rajomon_init /rajomon_init
COPY --from=builder /app/dagor_init /dagor_init
EXPOSE 2000
CMD ["./main"]

FROM alpine:latest AS svc-ms-9105
WORKDIR /root/
COPY --from=builder /out/ms-9105/main .
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

