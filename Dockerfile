# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/api \
    ./cmd/api

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/mock-email \
    ./cmd/mock-email

FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates

WORKDIR /app

FROM runtime AS api

COPY --from=builder /out/api /app/api
COPY configs/config.example.yaml /app/configs/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/api"]

FROM runtime AS mock-email

COPY --from=builder /out/mock-email /app/mock-email

EXPOSE 8081

ENTRYPOINT ["/app/mock-email"]
