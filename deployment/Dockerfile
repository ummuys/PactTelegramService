# ---------- build stage ----------
FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-s -w" \
  -o /out/telegram-service ./services/telegram/cmd

# ---------- runtime stage ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/telegram-service ./telegram-service

EXPOSE 50051
ENTRYPOINT ["./telegram-service"]