FROM alpine:3.8

RUN apk upgrade --update --no-cache

USER nobody

ADD build/_output/bin/aws-secret-operator /usr/local/bin/aws-secret-operator
