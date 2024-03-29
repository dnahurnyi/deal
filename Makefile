#
# Copyright 2019
#
# @author: Denys Nahurnyi
# @email:  dnahurnyi@gmail.com
# ---------------------------------------------------------------------------
SHELL = /bin/zsh

include build/common.mk

.PHONY = module unit-test module-install clean

TOPDIR := $(shell git rev-parse --show-toplevel)

module:
	@mkdir -p $(TOPDIR)/pb/generated
	@protoc -I $(TOPDIR)/pb/ -I $(TOPDIR)/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis/ $(TOPDIR)/common/pb/*.proto --go_out=plugins=grpc:$(TOPDIR)/common/pb/generated
	@protoc -I $(TOPDIR)/pb/ -I $(TOPDIR)/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis/ $(TOPDIR)/common/pb/*.proto --grpc-gateway_out=logtostderr=true:$(TOPDIR)/common/pb/generated
	@protoc -I $(TOPDIR)/pb/ -I $(TOPDIR)/vendor/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis/ $(TOPDIR)/common/pb/*.proto --swagger_out=logtostderr=true:$(TOPDIR)/common/pb/generated
	# @protoc-go-inject-tag -input=$(TOPDIR)/pb/generated/tenantMgr.pb.go
unit-test:

# install:
	# @go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
	# @go get -u google.golang.org/grpc
	# @go get -u google/api

proto:
	@echo Compiling proto files...
	@mkdir -p $(TOPDIR)/pb/generated
	@find **/*.proto -exec protoc \
		-I. -I$(GOPATH)/src -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:./pb/generated {} \;
	@find **/*.proto -exec protoc \
		-I. -I$(GOPATH)/src -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--grpc-gateway_out=logtostderr=true:./pb/generated {} \;
	@echo Done!

module-install:
	
clean:
	@rm -rf $(TOPDIR)/pb/generated

prepare:
	@echo Install needed deps and config environment
	@go get -u github.com/golang/protobuf/protoc-gen-go
	@kubectl config use-context gke_travel2-232717_us-central1-a_standard-cluster-1

return:
	@echo Return to the working environment environment
	@cd /Users/n826/ws/src/github.com/orkusinc/api && go install ./vendor/github.com/golang/protobuf/protoc-gen-go && cd /Users/n826/ws/src/github.com/DenysNahurnyi/deal
	@kubectl config use-context minikube