FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
ARG SERVICE
RUN if [ -z "$SERVICE" ]; then echo "SERVICE argument not set" && exit 1; fi
RUN cd ${SERVICE} && go build -o /app/main .

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .
# Copy static files for frontend if needed (frontend has static folder)
COPY --from=builder /app/frontend/static ./static

EXPOSE 8080

CMD ["./main"]
