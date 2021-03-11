# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

ROOTDIR="$(cd "$(dirname "$0")/.." ; pwd -P)"
${ROOTDIR}/cicd-scripts/e2e-tests-setup.sh -a install
if [ $? -ne 0 ]; then
    echo "Cannot setup environment successfully."
    exit 1
fi

# run test cases
${ROOTDIR}/cicd-scripts/tests.sh
if [ $? -ne 0 ]; then
    echo "Cannot pass all test cases."
    cat ${ROOTDIR}/pkg/tests/results.xml
    exit 1
fi