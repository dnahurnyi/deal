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
REPO  := n826/travel
TAG       := v1
DEPLOYMENT_YAML_TEMPL = $(TOPDIR)/kubernetes/templates/deployment.yaml.template
SVC_YAML_TEMPL = $(TOPDIR)/kubernetes/configs/minikube/svc.yaml.template

HTTPPORT := 8011
# NODEHTTPPORT := 30093
GRPCPORT := 8010
# NODEGRPCPORT := 30094

IV := 1

OS := $(shell uname -s)

DEPLOYMENT_YAML = $(TOPDIR)/kubernetes/generated/deployment-$(SERVICENAME).yaml
SVC_YAML = $(TOPDIR)/kubernetes/configs/generated/svc-$(SERVICENAME).yaml

module: build-docker

redep:
	@echo Redeploy $(SERVICENAME)
	@sed -e "s/{SERVICENAME}/${SERVICENAME}/g" < $(DEPLOYMENT_YAML_TEMPL) > $(DEPLOYMENT_YAML)
	@sed -i '' "s/{HTTPPORT}/$(HTTPPORT)/g" $(DEPLOYMENT_YAML)
	@sed -i '' "s/{GRPCPORT}/$(GRPCPORT)/g" $(DEPLOYMENT_YAML)
	@sed -i '' "s/{IMAGEVERSION}/$(IV)/g" $(DEPLOYMENT_YAML)

	@GOOS="linux" go build -o $(SERVICENAME) main.go
	@docker build --rm -t $(REPO):$(SERVICENAME)-v$(IV) . && rm $(SERVICENAME)
	@docker push $(REPO):$(SERVICENAME)-v$(IV)
	@kubectl delete deployment $(SERVICENAME)-deployment
	@kubectl create -f $(DEPLOYMENT_YAML)
	
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


