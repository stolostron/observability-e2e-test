#!/bin/bash
# Copyright (c) 2020 Red Hat, Inc.

# Required KUBECONFIG environment variable to run this script:

# set -eo pipefail

function usage() {
  echo "${0} -a ACTION [-i IMAGE]"
  echo ''
  # shellcheck disable=SC2016
  echo '  -a: Specifies the ACTION name, required, the value could be "install" or "uninstall".'
  # shellcheck disable=SC2016
  echo '  -i: Specifies the desired IMAGE, optional, the support image includes:
        quay.io/open-cluster-management/multicluster-observability-operator:<tag>
        quay.io/open-cluster-management/rbac-query-proxy:<tag>
        quay.io/open-cluster-management/metrics-collector:<tag>
        quay.io/open-cluster-management/endpoint-monitoring-operator:<tag>'
  echo ''
}

# Allow command-line args to override the defaults.
while getopts ":a:i:h" opt; do
  case ${opt} in
    a)
      ACTION=${OPTARG}
      ;;
    i)
      IMAGE=${OPTARG}
      ;;
    h)
      usage
      exit 0
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      usage
      exit 1
      ;;
  esac
done

if [[ -z "${ACTION}" ]]; then
  echo "Error: ACTION (-a) must be specified!"
  usage
  exit 1
fi

if [[ -z "${KUBECONFIG}" ]]; then
  echo "Error: environment variable KUBECONFIG must be specified!"
  exit 1
fi

TARGET_OS="$(uname)"
XARGS_FLAGS="-r"
SED_COMMAND='sed -i-e -e'
if [[ "$(uname)" == "Linux" ]]; then
    TARGET_OS=linux
elif [[ "$(uname)" == "Darwin" ]]; then
    TARGET_OS=darwin
    XARGS_FLAGS=
    SED_COMMAND='sed -i '-e' -e'
else
    echo "This system's OS $(TARGET_OS) isn't recognized/supported" && exit 1
fi

# Create bin directory and add it to PATH
mkdir -p ${HOME}/bin
export PATH=${PATH}:${HOME}/bin

ROOTDIR="$(cd "$(dirname "$0")/.." ; pwd -P)"
OBSERVABILITY_NS="open-cluster-management-observability"
OBSERVABILITYG_ADDON_NS="open-cluster-management-addon-observability"
OCM_DEFAULT_NS="open-cluster-management"
AGENT_NS="open-cluster-management-agent"
HUB_NS="open-cluster-management-hub"

COMPONENTS="multicluster-observability-operator rbac-query-proxy metrics-collector endpoint-monitoring-operator grafana-dashboard-loader"
COMPONENT_REPO="quay.io/open-cluster-management"

# Use snapshot for target release. Use latest one if no branch info detected, or not a release branch
BRANCH=""
LATEST_SNAPSHOT=""
if [[ "${PULL_BASE_REF}" == "release-"* ]]; then
    BRANCH=${PULL_BASE_REF#"release-"}
    BRANCH=${BRANCH}".0"
    LATEST_SNAPSHOT=`curl https://quay.io/api/v1/repository/open-cluster-management/multicluster-observability-operator | jq '.tags|with_entries(select(.key|contains("'${BRANCH}'-SNAPSHOT")))|keys[length-1]'`
fi
if [[ "${LATEST_SNAPSHOT}" == "null" ]] || [[ "${LATEST_SNAPSHOT}" == "" ]]; then
    LATEST_SNAPSHOT=`curl https://quay.io/api/v1/repository/open-cluster-management/multicluster-observability-operator | jq '.tags|with_entries(select(.key|contains("SNAPSHOT")))|keys[length-1]'`
fi

# trim the leading and tailing quotes
LATEST_SNAPSHOT="${LATEST_SNAPSHOT#\"}"
LATEST_SNAPSHOT="${LATEST_SNAPSHOT%\"}"
echo "The e2e test will use the image with tag: ${LATEST_SNAPSHOT}"

setup_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        echo "This script will install kubectl (https://kubernetes.io/docs/tasks/tools/install-kubectl/) on your machine"
        if [[ "$(uname)" == "Linux" ]]; then
            curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/linux/amd64/kubectl
        elif [[ "$(uname)" == "Darwin" ]]; then
            curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.0/bin/darwin/amd64/kubectl
        fi
        chmod +x ./kubectl && mv ./kubectl ${HOME}/bin/kubectl
    fi
}

setup_jq() {
    if ! command -v jq &> /dev/null; then
        if [[ "$(uname)" == "Linux" ]]; then
            curl -o jq -L https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64
        elif [[ "$(uname)" == "Darwin" ]]; then
            curl -o jq -L https://github.com/stedolan/jq/releases/download/jq-1.6/jq-osx-amd64
        fi
        chmod +x ./jq && mv ./jq ${HOME}/bin/jq
    fi
}

deploy_cert_manager() {
    if kubectl apply -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/cert-manager/cert-manager-openshift.yaml ; then
        echo "cert-manager was successfully deployed"
    else
        echo "Failed to deploy cert-manager"
        exit 1
    fi
}

delete_cert_manager() {
    kubectl delete -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/cert-manager/cert-manager-openshift.yaml
}

deploy_hub_spoke_core() {
    cd ${ROOTDIR}
    latest_release_branch=`git ls-remote --heads https://github.com/open-cluster-management/registration-operator.git release\* | tail -1 | cut -f 2 | cut -d '/' -f 3`
    git clone --depth 1 -b ${latest_release_branch} https://github.com/open-cluster-management/registration-operator.git && cd registration-operator

    # deploy hub and spoke via OLM
    make deploy
    # wait until cluster-manager deployments are ready
    kubectl -n $HUB_NS wait --timeout=240s --for=condition=Available deploy cluster-manager-registration-controller cluster-manager-registration-webhook cluster-manager-work-webhook
    # wait until klusterlet deployments are ready
    kubectl -n $AGENT_NS wait --timeout=240s --for=condition=Available deploy klusterlet-registration-agent klusterlet-work-agent
}

delete_hub_spoke_core() {
    cd ${ROOTDIR}/registration-operator
    # uninstall hub and spoke via OLM
    make clean-deploy
    rm -rf ${ROOTDIR}/registration-operator
    oc delete ns ${OCM_DEFAULT_NS} --ignore-not-found
}

approve_csr_joinrequest() {
    n=1
    while true
    do
        # TODO(morvencao): remove the hard-coded cluster label
        csr=`kubectl get csr -lopen-cluster-management.io/cluster-name=cluster1`
        if [[ ! -z $csr ]]; then
            csrnames=`kubectl get csr -lopen-cluster-management.io/cluster-name=cluster1 -o jsonpath={.items..metadata.name}`
            for csrname in ${csrnames}; do
                echo "Approve CSR: $csrname"
                kubectl certificate approve $csrname
            done
            break
        fi
        if [[ $n -ge 100 ]]; then
            print_mco_operator_log
            exit 1
        fi
        n=$((n+1))
        echo "Retrying in 10s..."
        sleep 10
    done

    n=1
    while true
    do
        cluster=`kubectl get managedcluster`
        if [[ ! -z $cluster ]]; then
            clusternames=`kubectl get managedcluster -o jsonpath={.items..metadata.name}`
            for clustername in ${clusternames}; do
                echo "Approve joinrequest for $clustername"
                kubectl patch managedcluster $clustername --patch '{"spec":{"hubAcceptsClient":true}}' --type=merge
            done
            break
        fi
        if [[ $n -ge 20 ]]; then
            print_mco_operator_log
            exit 1
        fi
        n=$((n+1))
        echo "Retrying in 5s..."
        sleep 5
    done
}

delete_csr() {
    # TODO(morvencao): remove the hard-coded cluster label
    kubectl delete csr -lopen-cluster-management.io/cluster-name=cluster1
}

print_mco_operator_log() {
    kubectl -n $DEFAULT_NS logs deploy/multicluster-observability-operator
}

deploy_mco_operator() {
    cd ${ROOTDIR}
    component_name=""
    if [[ ! -z "$1" ]]; then
        for comp in ${COMPONENTS}; do
            if [[ "$1" == *"$comp"* ]]; then
                component_name=$comp
                break
            fi
        done
        if [[ $component_name == "multicluster-observability-operator" ]]; then
            # copy the current multicluster-observability-operator commits to ROOTDIR for testing
            cp -r ${ROOTDIR}/../multicluster-observability-operator ${ROOTDIR}
            cd multicluster-observability-operator/
            $SED_COMMAND "s~image:.*$~image: $1~g" deploy/operator.yaml
        else
            git clone --depth 1 https://github.com/open-cluster-management/multicluster-observability-operator.git
            cd multicluster-observability-operator/
            # use latest snapshot for mco operator
            $SED_COMMAND "s~image:.*$~image: $COMPONENT_REPO/multicluster-observability-operator:$LATEST_SNAPSHOT~g" deploy/operator.yaml
            # test the concrete component
            component_anno_name=`echo $component_name | sed 's/-/_/g'`
            sed -i "/annotations.*/a \ \ \ \ mco-$component_anno_name-tag: ${1##*:}" ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/multiclusterobservability_cr.yaml
        fi
    else
        git clone --depth 1 https://github.com/open-cluster-management/multicluster-observability-operator.git
        cd multicluster-observability-operator/
        $SED_COMMAND "s~image:.*$~image: $COMPONENT_REPO/multicluster-observability-operator:$LATEST_SNAPSHOT~g" deploy/operator.yaml
    fi
    # Add mco-imageTagSuffix annotation
    $SED_COMMAND "s~mco-imageTagSuffix:.*~mco-imageTagSuffix: $LATEST_SNAPSHOT~g" ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/multiclusterobservability_cr.yaml

    # Install the multicluster-observability-operator
    kubectl create ns ${OBSERVABILITY_NS} || true
    # create api route
    app_domain=`kubectl -n openshift-ingress-operator get ingresscontrollers default -o jsonpath='{.status.domain}'`
    $SED_COMMAND "s~host: observatorium-api$~host: observatorium-api.$app_domain~g" ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/api-route.yaml
    kubectl -n ${OBSERVABILITY_NS} apply -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/api-route.yaml
    # create mco operator
    kubectl apply -f deploy/crds/observability.open-cluster-management.io_multiclusterobservabilities_crd.yaml
    kubectl apply -f deploy/req_crds
    kubectl apply -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/req_crds
    kubectl -n ${OBSERVABILITY_NS} apply -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/minio
    sleep 4
    kubectl create ns ${OCM_DEFAULT_NS} || true
    kubectl -n ${OCM_DEFAULT_NS} apply -f deploy
    kubectl -n ${OBSERVABILITY_NS} apply -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/multiclusterobservability_cr.yaml
}

delete_mco_operator() {
    cd ${ROOTDIR}/multicluster-observability-operator
    kubectl -n ${OBSERVABILITY_NS} delete -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/multiclusterobservability_cr.yaml
    # wait until all resources are deleted
    # TODO(morvencao): remove the hard-coded wait time
    sleep 60
    kubectl -n ${OCM_DEFAULT_NS} delete -f deploy
    kubectl -n ${OBSERVABILITY_NS} delete -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/minio
    kubectl delete -f deploy/crds/observability.open-cluster-management.io_multiclusterobservabilities_crd.yaml
    kubectl -n ${OBSERVABILITY_NS} delete -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/api-route.yaml
    kubectl delete ns ${OBSERVABILITY_NS}
    rm -rf ${ROOTDIR}/multicluster-observability-operator
}

# deploy the new grafana to check the dashboards from browsers
deploy_grafana_test() {
    cd ${ROOTDIR}/multicluster-observability-operator
    $SED_COMMAND "s~name: grafana$~name: grafana-test~g; s~app: multicluster-observability-grafana$~app: multicluster-observability-grafana-test~g; s~secretName: grafana-config$~secretName: grafana-config-test~g; s~secretName: grafana-datasources$~secretName: grafana-datasources-test~g; /MULTICLUSTEROBSERVABILITY_CR_NAME/d" manifests/base/grafana/deployment.yaml
    # replace with latest grafana-dashboard-loader image
    if [[ ! -z "$1" ]]; then
        if [[ "$1" == *"grafana-dashboard-loader"* ]]; then
            $SED_COMMAND "s~image: quay.io/open-cluster-management/grafana-dashboard-loader.*$~image: $1~g" manifests/base/grafana/deployment.yaml
        else
            $SED_COMMAND "s~image: quay.io/open-cluster-management/grafana-dashboard-loader.*$~image: $COMPONENT_REPO/grafana-dashboard-loader:$LATEST_SNAPSHOT~g" manifests/base/grafana/deployment.yaml
        fi
    else
        $SED_COMMAND "s~image: quay.io/open-cluster-management/grafana-dashboard-loader.*$~image: $COMPONENT_REPO/grafana-dashboard-loader:$LATEST_SNAPSHOT~g" manifests/base/grafana/deployment.yaml
    fi
    $SED_COMMAND "s~name: grafana$~name: grafana-test~g; s~app: multicluster-observability-grafana$~app: multicluster-observability-grafana-test~g" manifests/base/grafana/service.yaml
    $SED_COMMAND "s~namespace: open-cluster-management$~namespace: open-cluster-management-observability~g" manifests/base/grafana/deployment.yaml manifests/base/grafana/service.yaml

    kubectl -n ${OBSERVABILITY_NS} apply -f manifests/base/grafana/deployment.yaml
    kubectl -n ${OBSERVABILITY_NS} apply -f manifests/base/grafana/service.yaml

    # set up dedicated host for grafana-test
    app_domain=`kubectl -n openshift-ingress-operator get ingresscontrollers default -o jsonpath='{.status.domain}'`
    $SED_COMMAND "s~host: grafana-test$~host: grafana-test.$app_domain~g" ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/grafana/grafana-route-test.yaml
    kubectl -n ${OBSERVABILITY_NS} apply -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/grafana
}

# delete the grafana test
delete_grafana_test() {
    cd ${ROOTDIR}/multicluster-observability-operator
    kubectl -n ${OBSERVABILITY_NS} delete -f manifests/base/grafana/service.yaml
    kubectl -n ${OBSERVABILITY_NS} delete -f manifests/base/grafana/deployment.yaml
    kubectl -n ${OBSERVABILITY_NS} delete -f ${ROOTDIR}/cicd-scripts/e2e-setup-manifests/grafana
}

patch_placement_rule() {
    # Workaround for placementrules operator
    echo "Patch observability placementrule"
    n=1
    while true
    do
        if kubectl -n ${OBSERVABILITY_NS} get placementrule observability &> /dev/null; then
            break
        fi

        if [[ $n -ge 100 ]]; then
            print_mco_operator_log
            exit 1
        fi
        n=$((n+1))
        echo "Retrying in 10s..."
        sleep 10
    done

    # dump kubeconfig to local disk
    kubectl config view --raw --minify > ${HOME}/.kube/kubeconfig-hub
    cat ${HOME}/.kube/kubeconfig-hub|grep certificate-authority-data|awk '{split($0, a, ": "); print a[2]}'|base64 -d  >> ca
    cat ${HOME}/.kube/kubeconfig-hub|grep client-certificate-data|awk '{split($0, a, ": "); print a[2]}'|base64 -d >> crt
    cat ${HOME}/.kube/kubeconfig-hub|grep client-key-data|awk '{split($0, a, ": "); print a[2]}'|base64 -d >> key
    SERVER=$(cat ${HOME}/.kube/kubeconfig-hub|grep server|awk '{split($0, a, ": "); print a[2]}')
    curl --cert ./crt --key ./key --cacert ./ca -X PATCH -H "Content-Type:application/merge-patch+json" \
        $SERVER/apis/apps.open-cluster-management.io/v1/namespaces/$OBSERVABILITY_NS/placementrules/observability/status \
        -d @${ROOTDIR}/cicd-scripts/e2e-setup-manifests/templates/status.json
    rm -f ca crt key
}

wait_for_all_service_ready() {
    echo "Wait for all services are starting and runing..."
    n=1
    while true
    do
        if kubectl get ns ${OBSERVABILITYG_ADDON_NS} &> /dev/null; then
            if kubectl -n ${OBSERVABILITYG_ADDON_NS} get deploy endpoint-observability-operator metrics-collector-deployment &> /dev/null; then
                echo "Wait for all deploy endpoint-observability-operator metrics-collector-deployment are ready..."
                kubectl -n ${OBSERVABILITYG_ADDON_NS} wait --timeout=240s --for=condition=Available deploy endpoint-observability-operator metrics-collector-deployment
                break
            fi
        fi

        if [[ $n -ge 100 ]]; then
            print_mco_operator_log
            exit 1
        fi
        n=$((n+1))
        echo "Retrying in 10s..."
        sleep 10
    done
}

# function execute is the main routine to do the actual work
execute() {
    setup_kubectl
    setup_jq
    if [[ "${ACTION}" == "install" ]]; then
        deploy_cert_manager
        deploy_hub_spoke_core
        approve_csr_joinrequest
        deploy_mco_operator "${IMAGE}"
        deploy_grafana_test "${IMAGE}"
        patch_placement_rule
        wait_for_all_service_ready
        echo "OCM and Observability are installed successfuly..."
    elif [[ "${ACTION}" == "uninstall" ]]; then
        delete_grafana_test
        delete_mco_operator
        delete_hub_spoke_core
        delete_csr
        delete_cert_manager
        echo "OCM and Observability are uninstalled successfuly..."
    else
        echo "This ACTION ${ACTION} isn't recognized/supported" && exit 1
    fi
}

# start executing the ACTION
execute
