#
# Copyright 2018 Orkus, Inc
# All Rights Reserved.
#
# @author: Denys Nahurnyi.
# @email:  denys.nahurnyi@Blackthorn-vision.com
############################################################################
SHELL = /bin/bash

.PHONY = create-svc redep

TOPDIR := $(shell git rev-parse --show-toplevel)

SERVICENAME  := watchersvc
REPO  := n826/travel
TAG       := v1
DEPLOYMENT_YAML_TEMPL = $(TOPDIR)/kubernetes/templates/deployment.yaml.template
SVC_YAML_TEMPL = $(TOPDIR)/kubernetes/templates/svc.yaml.template

HTTPPORT := 8015
NODEHTTPPORT := 30255
GRPCPORT := 8014
NODEGRPCPORT := 30254

IV := 1

OS := $(shell uname -s)

DEPLOYMENT_YAML = $(TOPDIR)/kubernetes/generated/deployment-$(SERVICENAME).yaml
SVC_YAML = $(TOPDIR)/kubernetes/generated/svc-$(SERVICENAME).yaml

create-svc:
	@sed -e "s/{SERVICENAME}/${SERVICENAME}/g" < $(SVC_YAML_TEMPL) > $(SVC_YAML)
	@sed -i '' "s/{HTTPPORT}/$(HTTPPORT)/g" $(SVC_YAML)
	@sed -i '' "s/{GRPCPORT}/$(GRPCPORT)/g" $(SVC_YAML)
	@sed -i '' "s/{NODEHTTPPORT}/$(NODEHTTPPORT)/g" $(SVC_YAML)
	@sed -i '' "s/{NODEGRPCPORT}/$(NODEGRPCPORT)/g" $(SVC_YAML)

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

first-dep: 
	@echo First deployment of $(SERVICENAME)
	@make create-svc
	@make redep