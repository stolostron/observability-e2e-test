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
echo "To sleep 180s"
sleep 180
kubectl get pod -A
kubectl get deployment -n open-cluster-management-addon-observability endpoint-observability-operator -o yaml
POD_NAME=$(kubectl get po -n open-cluster-management-addon-observability|grep endpoint| awk '{split($0, a, " "); print a[1]}')
echo $POD_NAME
kubectl logs -n open-cluster-management-addon-observability $POD_NAME -c endpoint-observability-operator
POD_NAME=$(kubectl get po -n open-cluster-management-addon-observability|grep metrics-collector| awk '{split($0, a, " "); print a[1]}')
echo $POD_NAME
kubectl logs -n open-cluster-management-addon-observability $POD_NAME -c endpoint-observability-operator

kubectl get observabilityaddon observability-addon -n open-cluster-management-addon-observability -o yaml
kubectl get clusterrolebinding -A -o yaml|grep "name: klusterlet-work-sa"
kubectl get clusterrolebinding -A|grep "open-cluster-management:klusterlet-work:agent"
kubectl get clusterrolebinding open-cluster-management:klusterlet-work:agent -o yaml

export KUBECONFIG=$HOME/.kube/kind-config-hub
kubectl get crd|grep observability
kubectl get manifestwork -A
kubectl get manifestwork -n cluster1 cluster1-observability-operator -o yaml
kubectl get manifestwork -n cluster1 cluster1-observability-operator-res -o yaml
kubectl get mco observability -o yaml

ginkgo -debug -trace -v ./pkg/tests -- -options=../../resources/options.yaml -v=3

cat ./pkg/tests/results.xml | grep failures=\"0\" | grep errors=\"0\"
if [ $? -ne 0 ]; then
    exit 1
fi