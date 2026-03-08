FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/. ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bootstrap-admin ./cmd/bootstrap-admin

FROM alpine:3.22
RUN apk add --no-cache ca-certificates postgresql17-client
WORKDIR /app

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/bootstrap-admin /usr/local/bin/bootstrap-admin
COPY backend/migrations /app/backend/migrations
COPY scripts/apply-migrations.sh /usr/local/bin/apply-migrations.sh
COPY scripts/dev-seed.sh /usr/local/bin/dev-seed.sh

ENTRYPOINT ["/usr/local/bin/api"]
