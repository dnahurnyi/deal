#
# Copyright 2018 Orkus, Inc
# All Rights Reserved.
#
# @author: Denys Nahurnyi.
# @email:  denys.nahurnyi@Blackthorn-vision.com
############################################################################
SHELL = /bin/bash

.PHONY = module build-docker push clean

TOPDIR := $(shell git rev-parse --show-toplevel)

SERVICENAME  := datasvc
TAG       := v1
ifneq ($(BUILD_TYPE),PRODUCTION)
REPO_URI  := n826
DEPLOYMENT_YAML_TEMPL = $(TOPDIR)/kubernetes/configs/minikube/deployment.yaml.template
SVC_YAML_TEMPL = $(TOPDIR)/kubernetes/configs/minikube/svc.yaml.template
else
REPO_URI := ${ECR_URI}
SVC_YAML_TEMPL = $(TOPDIR)/kubernetes/configs/prod/svc.yaml.template
DEPLOYMENT_YAML_TEMPL = $(TOPDIR)/kubernetes/configs/prod/deployment.yaml.template
endif

HTTPPORT := 8093
NODEHTTPPORT := 30093
GRPCPORT := 8094
NODEGRPCPORT := 30094

OS := $(shell uname -s)

DEPLOYMENT_YAML = $(TOPDIR)/kubernetes/configs/generated/deployment-$(SERVICENAME).yaml

SVC_YAML = $(TOPDIR)/kubernetes/configs/generated/svc-$(SERVICENAME).yaml

module: build-docker

build-docker: $(DEPLOYMENT_YAML_TEMPL)
	@$(MAKE) -C $(TOPDIR) gen_kubernetes_templates
	@GOOS="linux" go build -o $(SERVICENAME) $(TOPDIR)/svc/dataSvc/main.go
	@docker build  --rm -t $(SERVICENAME):$(TAG) .
	@sed -e "s/{USER}/${USER}/g" < $(DEPLOYMENT_YAML_TEMPL) > $(DEPLOYMENT_YAML)
ifeq ($(OS),Darwin) 
	@sed -i '' "s/{VERSION}/$(TAG)/g" $(DEPLOYMENT_YAML)
	@sed -i '' "s/{SERVICENAME}/$(SERVICENAME)/g" $(DEPLOYMENT_YAML)
	@sed -i '' "s/{HTTPPORT}/$(HTTPPORT)/g" $(DEPLOYMENT_YAML)
	@sed -i '' "s/{GRPCPORT}/$(GRPCPORT)/g" $(DEPLOYMENT_YAML)
	@sed -i '' "s/{BUILD_TYPE}/${BUILD_TYPE}/g" $(DEPLOYMENT_YAML)
	@sed -e "s/{SERVICENAME}/${SERVICENAME}/g" < $(SVC_YAML_TEMPL) > $(SVC_YAML)
	@sed -i '' "s/{HTTPPORT}/$(HTTPPORT)/g" $(SVC_YAML)
	@sed -i '' "s/{GRPCPORT}/$(GRPCPORT)/g" $(SVC_YAML)
	@sed -i '' "s/{NODEHTTPPORT}/$(NODEHTTPPORT)/g" $(SVC_YAML)
	@sed -i '' "s/{NODEGRPCPORT}/$(NODEGRPCPORT)/g" $(SVC_YAML)
else
	@sed -ie "s/{VERSION}/$(TAG)/g" $(DEPLOYMENT_YAML)
	@sed -ie "s/{SERVICENAME}/$(SERVICENAME)/g" $(DEPLOYMENT_YAML)
	@sed -ie "s/{HTTPPORT}/$(HTTPPORT)/g" $(DEPLOYMENT_YAML)
	@sed -ie "s/{GRPCPORT}/$(GRPCPORT)/g" $(DEPLOYMENT_YAML)
	@sed -ie "s/{BUILD_TYPE}/${BUILD_TYPE}/g" $(DEPLOYMENT_YAML)
	@sed -e "s/{SERVICENAME}/${SERVICENAME}/g" < $(SVC_YAML_TEMPL) > $(SVC_YAML)
	@sed -ie "s/{HTTPPORT}/$(HTTPPORT)/g" $(SVC_YAML)
	@sed -ie "s/{GRPCPORT}/$(GRPCPORT)/g" $(SVC_YAML)
	@sed -ie "s/{NODEHTTPPORT}/$(NODEHTTPPORT)/g" $(SVC_YAML)
	@sed -ie "s/{NODEGRPCPORT}/$(NODEGRPCPORT)/g" $(SVC_YAML)
endif
	@rm $(SERVICENAME) 
ifeq ($(BUILD_TYPE),PRODUCTION)
	$(MAKE) push
endif

module-install: module
ifneq ($(BUILD_TYPE),PRODUCTION)
	@-kubectl delete deployment ${SERVICENAME}-deployment
else
	@-kubectl delete deployment ${SERVICENAME}-deployment -n nginx-ingress
endif
	@kubectl create -f $(TOPDIR)/kubernetes/configs/generated/deployment-$(SERVICENAME).yaml
	@kubectl apply -f $(TOPDIR)/kubernetes/configs/generated/svc-$(SERVICENAME).yaml

push: 
	@echo docker push $(SERVICENAME)
	@docker tag $(SERVICENAME):$(TAG) $(REPO_URI)/$(SERVICENAME):$(TAG)
	@docker push  $(REPO_URI)/$(SERVICENAME):$(TAG)

unit-test:
	@go test -v "github.com/orkusinc/api/tenantMgr"

clean:
	@echo clean docker images
	@-rm $(DEPLOYMENT_YAML)
	@-rm $(SVC_YAML)
ifneq ($(BUILD_TYPE),PRODUCTION)
	@-kubectl delete deployment ${SERVICENAME}-deployment
	@-kubectl delete svc ${SERVICENAME}-service
else
	@-kubectl delete deployment ${SERVICENAME}-deployment -n nginx-ingress
	@-kubectl delete svc ${SERVICENAME}-service -n nginx-ingress
endif


