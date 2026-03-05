FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN go mod tidy
ARG SERVICE
RUN go build -o /app/main ./${SERVICE}

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 2000
CMD ["./main"]
