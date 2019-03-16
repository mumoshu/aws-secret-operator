IMAGE ?= mumoshu/aws-secret-operator:canary

publish:
	operator-sdk build $(IMAGE) && docker push $(IMAGE)

install-tools:
	go get github.com/aws/aws-sdk-go/aws/session
	go get github.com/aws/aws-sdk-go/service/secretsmanager
	go get github.com/pkg/errors
