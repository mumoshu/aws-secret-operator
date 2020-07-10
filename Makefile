VERSION ?= canary
REPO ?= mumoshu/aws-secret-operator
IMAGE ?= $(REPO):$(VERSION)
GO ?= go1.14.4

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

.PHONY: e2e
e2e:
	kubectl create namespace operator-test || true
	$(GO) run github.com/operator-framework/operator-sdk/cmd/operator-sdk test local ./test/e2e --operator-namespace operator-test --up-local
