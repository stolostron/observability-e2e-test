# the tests.sh is designed to run in KinD cluster

go get -u github.com/onsi/ginkgo/ginkgo

export KUBECONFIG=$HOME/.kube/kind-config-hub
export IMPORT_KUBECONFIG=$HOME/.kube/kind-config-spoke
export SKIP_INSTALL_STEP=true

git clone https://github.com/open-cluster-management/observability-gitops.git

printf "options:" >> resources/options.yaml
printf "\n  hub:" >> resources/options.yaml
printf "\n    baseDomain: placeholder" >> resources/options.yaml
printf "\n    masterURL: https://127.0.0.1:32806" >> resources/options.yaml
printf "\n    grafanaURL: http://127.0.0.1" >> resources/options.yaml
printf "\n    grafanaHost: grafana-test" >> resources/options.yaml
printf "\n  clusters:" >> resources/options.yaml
printf "\n    - name: cluster1" >> resources/options.yaml
printf "\n      masterURL: https://127.0.0.1:32807" >> resources/options.yaml

# workaround to fix the manifestwork problem
cat >./tmp.yaml <<EOL
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: 'open-cluster-management:klusterlet-work:agent-addition1'
subjects:
  - kind: ServiceAccount
    name: klusterlet-work-sa
    namespace: open-cluster-management-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: 'open-cluster-management:klusterlet-work:agent1'

---

kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: 'open-cluster-management:klusterlet-work:agent1'
rules:
  - verbs:
      - get
      - list
      - watch
      - create
      - delete
      - update
    apiGroups:
      - observability.open-cluster-management.io
    resources:
      - observabilityaddons
EOL
export KUBECONFIG=$HOME/.kube/kind-config-spoke
kubectl apply -f ./tmp.yaml
export KUBECONFIG=$HOME/.kube/kind-config-hub

ginkgo -debug -trace -v ./pkg/tests -- -options=../../resources/options.yaml -v=3

cat ./pkg/tests/results.xml | grep failures=\"0\" | grep errors=\"0\"
if [ $? -ne 0 ]; then
    exit 1
fi