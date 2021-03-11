# Copyright Contributors to the Open Cluster Management project

-include /opt/build-harness/Makefile.prow

build:
	ginkgo build ./pkg/tests/

test-unit:
	@echo "Running Unit Tests.."	
