#
# Copyright 2019
#
# @author: Denys Nahurnyi
# @email:  dnahurnyi@gmail.com
# ---------------------------------------------------------------------------
SHELL = /bin/bash

TOPDIR := $(shell git rev-parse --show-toplevel)

.PHONY = gen_kubernetes_templates

gen_kubernetes_templates:
	@mkdir -p $(TOPDIR)/kubernetes/generated