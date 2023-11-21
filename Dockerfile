FROM golang:1.21.3-alpine AS builder

ENV GO111MODULE=on
RUN apk add --no-cache git build-base gcc musl-dev mercurial
WORKDIR /app
COPY . /app
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -trimpath -ldflags="-s -w" -o /gofireprox ./cmd/gofireprox
RUN rm -rf /app

FROM alpine:3.17.3
RUN apk -U upgrade --no-cache \
    && apk add --no-cache bind-tools ca-certificates
COPY --from=builder /gofireprox /usr/local/bin/

ENTRYPOINT ["gofireprox"]