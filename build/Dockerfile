FROM alpine:3.8

RUN apk add --update --no-cache ca-certificates

USER nobody

ADD build/_output/bin/aws-secret-operator /usr/local/bin/aws-secret-operator
