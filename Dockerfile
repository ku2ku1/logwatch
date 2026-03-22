FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o logwatch ./cmd/logwatch

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/logwatch .
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/data ./data

EXPOSE 8080
CMD ["./logwatch"]