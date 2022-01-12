# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

-include $(shell [ -f ".build-harness-bootstrap" ] || curl -sL -o .build-harness-bootstrap -H "Authorization: token $(GITHUB_TOKEN)" -H "Accept: application/vnd.github.v3.raw" "https://raw.github.com/stolostron/build-harness-extensions/main/templates/Makefile.build-harness-bootstrap"; echo .build-harness-bootstrap)
-include /opt/build-harness/Makefile.prow

build:
	ginkgo build ./pkg/tests/

test-unit:
	@echo "Running Unit Tests.."

test-e2e: test-e2e-setup
	@echo "Running E2E Tests.."
	@./cicd-scripts/run-e2e-tests.sh

test-e2e-setup:
	@echo "Seting up E2E Tests environment..."
ifdef COMPONENT_IMAGE_NAME
	# override the image for the e2e test
	@./cicd-scripts/setup-e2e-tests.sh -a install -i $(COMPONENT_IMAGE_NAME)
else
	# fall back to the latest snapshot image from quay.io for the e2e test
	@./cicd-scripts/setup-e2e-tests.sh -a install
endif

test-e2e-clean:
	@echo "Clean E2E Tests environment..."
ifdef COMPONENT_IMAGE_NAME
	@./cicd-scripts/setup-e2e-tests.sh -a uninstall -i $(COMPONENT_IMAGE_NAME)
else
	@./cicd-scripts/setup-e2e-tests.sh -a uninstall
endif
