FROM golang:1.26-alpine AS builder
WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/. ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bootstrap-admin ./cmd/bootstrap-admin
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/import-challenges ./cmd/import-challenges

FROM alpine:3.22
RUN apk add --no-cache ca-certificates postgresql17-client
WORKDIR /app

COPY --from=builder /out/api /usr/local/bin/api
COPY --from=builder /out/bootstrap-admin /usr/local/bin/bootstrap-admin
COPY --from=builder /out/import-challenges /usr/local/bin/import-challenges
COPY backend/migrations /app/backend/migrations
COPY challenges /app/challenges
COPY scripts/apply-migrations.sh /usr/local/bin/apply-migrations.sh
COPY scripts/dev-seed.sh /usr/local/bin/dev-seed.sh
COPY scripts/import-challenges.sh /usr/local/bin/import-challenges.sh

ENTRYPOINT ["/usr/local/bin/api"]
