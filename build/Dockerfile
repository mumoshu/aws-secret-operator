FROM golang:1.12 as builder

ARG APP_VERSION

ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /go/src/github.com/mumoshu/aws-secret-operator
COPY . /go/src/github.com/mumoshu/aws-secret-operator

RUN if [ -n "${APP_VERSION}" ]; then git checkout -b tag refs/tags/${APP_VERSION} || git checkout -b branch ${APP_VERSION}; fi \
    && make build -e GO111MODULE=on

FROM alpine:3.10

RUN apk add --update --no-cache ca-certificates libc6-compat

USER nobody

COPY --from=builder /go/src/github.com/mumoshu/aws-secret-operator/bin/aws-secret-operator /usr/local/bin/aws-secret-operator
