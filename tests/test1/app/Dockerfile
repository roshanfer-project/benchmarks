FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY app ./app
COPY utils ./utils

# Build the application
WORKDIR /app/app
RUN go build -o main .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/app/main .

EXPOSE 8080

CMD ["./main"]
