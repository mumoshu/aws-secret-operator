VERSION ?= canary
REPO ?= mumoshu/aws-secret-operator
IMAGE ?= $(REPO):$(VERSION)
GO ?= go

.PHONY: build
build:
	$(GO) build -o bin/aws-secret-operator .

.PHONY: image
image:
	docker build -t $(IMAGE) .

.PHONY: push
publish:
	docker push $(IMAGE)

crds: controller-gen
	$(CONTROLLER_GEN) crd rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=deploy/crds

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths="./..."

# Find or download controller-gen
#
# Note that controller-gen newer than 0.4.1 is needed for https://github.com/kubernetes-sigs/controller-tools/issues/444#issuecomment-680168439
# Otherwise we get errors like the below:
#   Error: failed to install CRD crds/actions.summerwind.dev_runnersets.yaml: CustomResourceDefinition.apiextensions.k8s.io "runnersets.actions.summerwind.dev" is invalid: [spec.validation.openAPIV3Schema.properties[spec].properties[template].properties[spec].properties[containers].items.properties[ports].items.properties[protocol].default: Required value: this property is in x-kubernetes-list-map-keys, so it must have a default or be a required property, spec.validation.openAPIV3Schema.properties[spec].properties[template].properties[spec].properties[initContainers].items.properties[ports].items.properties[protocol].default: Required value: this property is in x-kubernetes-list-map-keys, so it must have a default or be a required property]
#
# Note that controller-gen newer than 0.6.0 is needed due to https://github.com/kubernetes-sigs/controller-tools/issues/448
# Otherwise ObjectMeta embedded in Spec results in empty on the storage.
controller-gen:
ifeq (, $(shell which controller-gen))
ifeq (, $(wildcard $(GOBIN)/controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	$(GO) mod init tmp ;\
	$(GO) get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
endif
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
