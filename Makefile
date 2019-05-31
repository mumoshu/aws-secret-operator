VERSION ?= 0.2.0
IMAGE ?= mumoshu/aws-secret-operator:$(VERSION)

publish:
	operator-sdk build $(IMAGE) && docker push $(IMAGE)

install-tools:
	go get github.com/aws/aws-sdk-go/aws/session
	go get github.com/aws/aws-sdk-go/service/secretsmanager
	go get github.com/pkg/errors
