# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

#!/bin/bash
echo "BUILD GOES HERE!"

echo "<repo>/<component>:<tag> : $1"

git clone https://github.com/open-cluster-management/observability-gitops.git

export DOCKER_IMAGE_AND_TAG=${1}

make docker/build

