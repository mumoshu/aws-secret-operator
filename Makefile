VERSION ?= 0.2.4
REPO ?= mumoshu/aws-secret-operator
IMAGE ?= $(REPO):$(VERSION)

.PHONY: build
build:
	go build -o bin/aws-secret-operator ./cmd/manager

.PHONY: image
image:
	DOCKERFILE_PATH=./build/Dockerfile IMAGE_NAME=$(IMAGE) REPO=$(REPO) hooks/build

publish:
	operator-sdk build $(IMAGE) && docker push $(IMAGE)

install-tools:
	go get github.com/aws/aws-sdk-go@v1.25.10
	go get github.com/aws/aws-sdk-go/aws/session
	go get github.com/aws/aws-sdk-go/service/secretsmanager
	go get github.com/pkg/errors
