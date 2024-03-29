# Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
# or more contributor license agreements. Licensed under the Elastic License;
# you may not use this file except in compliance with the Elastic License.

#
# Generate single YAML to deploy the operator.
#

export SHELL := /bin/bash

ifndef NAMESPACE
$(error NAMESPACE not set)
endif
ifndef OPERATOR_IMAGE
$(error OPERATOR_IMAGE not set)
endif

generate-all-in-one:
	@ for yaml in $$(LC_COLLATE=C ls all-in-one/*yaml); do \
		printf "\n---\n" && cat $$yaml ; \
	done | \
	sed \
		-e "/# /d" \
		-e "s|<OPERATOR_IMAGE>|$$OPERATOR_IMAGE|g" \
		-e "s|<NAMESPACE>|$$NAMESPACE|g"

generate-global:
	@ for yaml in $$(LC_COLLATE=C ls global/*yaml); do \
		printf "\n---\n" && cat $$yaml ; \
	done | \
	sed \
		-e "/# /d" \
		-e "s|<OPERATOR_IMAGE>|$$OPERATOR_IMAGE|g" \
		-e "s|<NAMESPACE>|$$NAMESPACE|g"

generate-namespace:
ifndef MANAGED_NAMESPACE
	$(error MANAGED_NAMESPACE not set)
endif
	@ for yaml in $$(LC_COLLATE=C ls namespace/*yaml); do \
		printf "\n---\n" && cat $$yaml ; \
	done | \
	sed \
		-e "/# /d" \
		-e "s|<OPERATOR_IMAGE>|$$OPERATOR_IMAGE|g" \
		-e "s|<NAMESPACE>|$$NAMESPACE|g" \
		-e "s|<MANAGED_NAMESPACE>|$$MANAGED_NAMESPACE|g"
