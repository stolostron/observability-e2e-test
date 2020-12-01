WORKDIR=`pwd`
cd ${WORKDIR}/..
git clone https://github.com/open-cluster-management/observability-kind-cluster.git
cd observability-kind-cluster
./setup.sh
if [ $? -ne 0 ]; then
    echo "Cannot setup environment successfully."
    exit 1
fi

go get -u github.com/onsi/ginkgo/ginkgo

export KUBECONFIG=$HOME/.kube/kind-config-hub
export IMPORT_KUBECONFIG=$HOME/.kube/kind-config-spoke
export SKIP_INSTALL_STEP=true
export SKIP_UNINSTALL_STEP=true

cd ${WORKDIR}

git clone https://github.com/open-cluster-management/observability-gitops.git

printf "options:" >> resources/options.yaml
printf "\n  hub:" >> resources/options.yaml
printf "\n    baseDomain: placeholder" >> resources/options.yaml
printf "\n    masterURL: https://127.0.0.1:32806" >> resources/options.yaml
printf "\n    grafanaURL: http://127.0.0.1" >> resources/options.yaml
printf "\n    grafanaHost: grafana-test" >> resources/options.yaml
printf "\n  clusters:" >> resources/options.yaml
printf "\n    - name: spoke" >> resources/options.yaml
printf "\n      masterURL: https://127.0.0.1:32807" >> resources/options.yaml

ginkgo -v ./pkg/tests -- -options=../../resources/options.yaml -v=3

cat ./pkg/tests/results.xml | grep failures=\"0\" | grep errors=\"0\"
if [ $? -ne 0 ]; then
    echo "Cannot pass all test cases."
    cat results.xml
    exit 1
fi