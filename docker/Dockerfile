FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o onenote-mcp-server ./cmd/onenote-mcp-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/onenote-mcp-server .
ENTRYPOINT ["./onenote-mcp-server"] 