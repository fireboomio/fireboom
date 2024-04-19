FROM golang:1.22.0-alpine3.19 AS builder
WORKDIR /build
ENV GOPROXY https://goproxy.cn
COPY go.mod go.sum ./
RUN go mod tidy && apk add --no-cache git