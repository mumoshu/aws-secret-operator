IMAGE ?= mumoshu/aws-secret-operator:canary

publish:
	operator-sdk build $(IMAGE) && docker push $(IMAGE)
