FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o stackpilot .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/stackpilot .
EXPOSE 8089
ENTRYPOINT ["./stackpilot", "server", "--host", "0.0.0.0", "--port", "8089"]
