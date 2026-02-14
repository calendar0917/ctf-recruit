FROM golang:1.22-alpine

WORKDIR /app

ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY
ARG NO_PROXY
ARG GOPROXY=https://goproxy.cn,direct

ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV ALL_PROXY=${ALL_PROXY}
ENV NO_PROXY=${NO_PROXY}
ENV GOPROXY=${GOPROXY}

RUN apk add --no-cache ca-certificates git && update-ca-certificates

COPY go.mod go.sum ./
RUN go mod download -x

COPY . .

RUN go build -o /bin/api ./cmd/api

EXPOSE 8080

CMD ["/bin/api"]
