# observability-e2e-test

This is modeled after: https://github.com/open-cluster-management/open-cluster-management-e2e

This is a container which will be called from:

1. Canary Tests
2. Regular Build PRs

The tests in this container will:

1. Create the MCO CR . The Object store to be already in place for CR to work.
2. Wait for the installation to complete.
3. Then check the entire Observability suite (Hub and Addon) is working as expected including disable/enable, Grafana etc.

## Setup E2E Testing Environment

If you only have an OCP cluster and haven't installed Observability yet, then you can install the Observability (both Hub and Addon) by the following steps:

1. clone this repo:

```
$ git clone git@github.com:open-cluster-management/observability-e2e-test.git
```

2. export `KUBECONFIG` environment to the kubeconfig of your OCP cluster:

```
$ export KUBECONFIG=<kubeconfig-file-of-your-ocp-cluster>
```

3. Then simply execute the following command to install the Observability and its dependencies:

```
$ make test-e2e-setup
```

By default, the command will try to install the Observability and its dependencies with images of latest snapshot from `quay.io/repository/open-cluster-management` image registry. You may want to override the specified component image to test specified component is working with other components, you can simply implement that by exporting `COMPONENT_IMAGE_NAME` environment, for example, if you want to test metrics-collector image `quay.io/open-cluster-management/metrics-collector:test` with other images of latest snapshot, then execute the following command before running command in step 2:

```
$ COMPONENT_IMAGE_NAME=quay.io/open-cluster-management/metrics-collector:test
```

The supported component images include:

- multicluster-observability-operator
- rbac-query-proxy
- metrics-collector
- endpoint-monitoring-operator
- grafana-dashboard-loader

> Note: the component image override is useful when you want to test each stockholder repositories, you only need to export the `COMPONENT_IMAGE_NAME` environment if running the e2e testing locally. For the CICD pipeline, the prow will take care of entirely process, that means that when you raise a PR to the stockholder repositories, the prow will build the image based the source code of your commit and then install the Observability accordingly.

## Running E2E Testing

Before you run the E2E, make sure [ginkgo](https://github.com/onsi/ginkgo) is installed:

```
$ go get -u github.com/onsi/ginkgo/ginkgo
```

then, depending on how your E2E environment is set up, you can choose the following exclusive methods to run the e2e testing:

1. If your E2E environment is set up with steps in the last stanza, then simply run the e2e testing with the following command:

```
# make test-e2e
```

2. If you set up the E2E environment manually, then you need to copy `resources/options.yaml.template` to `resources/options.yaml`, and update values specific to your environment:

```
$ cp resources/options.yaml.template resources/options.yaml
$ cat resources/options.yaml
options:
  identityProvider: kube:admin
  hub:
    name: HUB_CLUSTER_NAME
    baseDomain: BASE_DOMAIN
```

(optional) If there is an imported cluster in the test environment, need to add the cluster info into `options.yaml`:

```
$ cat resources/options.yaml
options:
  identityProvider: kube:admin
  hub:
    name: HUB_CLUSTER_NAME
    baseDomain: BASE_DOMAIN
  clusters:
  - name: IMPORT_CLUSTER_NAME
    baseDomain: IMPORT_CLUSTER_BASE_DOMAIN
    kubecontext: IMPORT_CLUSTER_KUBE_CONTEXT
```

then start to run e2e testing manually by the following command:

```
$ export BUCKET=YOUR_S3_BUCKET
$ export REGION=YOUR_S3_REGION
$ export AWS_ACCESS_KEY_ID=YOUR_S3_AWS_ACCESS_KEY_ID
$ export AWS_SECRET_ACCESS_KEY=YOUR_S3_AWS_SECRET_ACCESS_KEY
$ export KUBECONFIG=~/.kube/config
$ ginkgo -v pkg/tests/ -- -options=../../resources/options.yaml -v=3
```

(optional) If there is an imported cluster in the test environment, need to set more environment.

```
$ export IMPORT_KUBECONFIG=~/.kube/import-cluster-config
```

## Running with Docker

1. clone this repo:

```
$ git clone git@github.com:open-cluster-management/observability-e2e-test.git
```

2. copy `resources/options.yaml.template` to `resources/options.yaml`, and update values specific to your environment:

```
$ cp resources/options.yaml.template resources/options.yaml
$ cat resources/options.yaml
options:
  identityProvider: kube:admin
  hub:
    name: HUB_CLUSTER_NAME
    baseDomain: BASE_DOMAIN
```
(optional)If there is an imported cluster in the test environment, need to add the cluster info into options.yaml
```
$ cat resources/options.yaml
options:
  identityProvider: kube:admin
  hub:
    name: HUB_CLUSTER_NAME
    baseDomain: BASE_DOMAIN
  clusters:
  - name: IMPORT_CLUSTER_NAME
    baseDomain: IMPORT_CLUSTER_BASE_DOMAIN
    kubecontext: IMPORT_CLUSTER_KUBE_CONTEXT 
```

3. copy `resources/env.list.template` to `resources/env.list`, and update values specific to your s3 configuration:

```
$ cp resources/env.list.template resources/env.list
$ cat resources/env.list
BUCKET=YOUR_S3_BUCKET
REGION=YOUR_S3_REGION
AWS_ACCESS_KEY_ID=YOUR_S3_AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY=YOUR_S3_AWS_SECRET_ACCESS_KEY
```

4. oc login to your cluster in which observability is installed - and make sure that remains the current-context in kubeconfig:

```
$ kubectl config current-context
open-cluster-management-observability/api-demo-dev05-red-chesterfield-com:6443/kube:admin
```

5. build docker image:

```
$ docker build -t observability-e2e-test:latest .
```

6. (optional) If there is an imported cluster in the test environment, need to copy its' kubeconfig file into as ~/.kube/ as import-kubeconfig

```
$ cp {IMPORT_CLUSTER_KUBE_CONFIG_PATH} ~/.kube/import-kubeconfig
```

7. run testing:

```
$ docker run -v ~/.kube/:/opt/.kube -v $(pwd)/results:/results -v $(pwd)/resources:/resources --env-file $(pwd)/resources/env.list observability-e2e-test:latest
```

In Canary environment, this is the container that will be run - and all the volumes etc will passed on while starting the docker container using a helper script.

## Contributing to E2E

### Options.yaml

The values in the options.yaml are optional values read in by E2E. If you do not set an option, the test case that depends on the option should skip the test. The sample values in the option.yaml.template should provide enough context for you fill in with the appropriate values. Further, in the section below, each test should document their test with some detail.

### Skip install and uninstall

For developing and testing purposes, you can set the following env to skip the install and uninstall steps to keep your current MCO instance.

- SKIP_INSTALL_STEP:  if set to `true`, the testing will skip the install step
- SKIP_UNINSTALL_STEP:  if set to `true`, the testing will skip the uninstall step

For example, run the following command will skip the install and uninstall step:

```
$ export SKIP_INSTALL_STEP=true
$ export SKIP_UNINSTALL_STEP=true
$ export BUCKET=YOUR_S3_BUCKET
$ export REGION=YOUR_S3_REGION
$ export AWS_ACCESS_KEY_ID=YOUR_S3_AWS_ACCESS_KEY_ID
$ export AWS_SECRET_ACCESS_KEY=YOUR_S3_AWS_SECRET_ACCESS_KEY
$ export KUBECONFIG=~/.kube/config
$ ginkgo -v -- -options=resources/options.yaml -v=3
```

### Focus Labels

* Each `It` specification should end with a label which helps automation segregate running of specs.
* The choice of labels is up to the contributor, with the one guideline, that the second label, be `g0-gN`, to indicate the `run level`, with `g0` denoting that this test runs within a few minutes, and `g5` denotes a testcase that will take > 30 minutes to complete. See examples below:

`	It("should have not the expected MCO addon pods (addon/g0)", func() {`

Examples:

```yaml
  It("should have the expected args in compact pod (reconcile/g0)", func() {
  It("should work in basic mode (reconcile/g0)", func() {
  It("should have not the expected MCO addon pods (addon/g0)", func() {
  It("should have not metric data (addon/g0)", func() {
  It("should be able to access the grafana console (grafana/g0)", func() {
  It("should have metric data in grafana console (grafana/g0)", func() {
    ....
```

* The `--focus` and `--skip` are ginkgo directives that allow you to choose what tests to run, by providing a REGEX express to match. Examples of using the focus:

  * `ginkgo --focus="g0"`
  * `ginkgo --focus="grafana/g0"`
  * `ginkgo --focus="addon"`

* To run with verbose ginkgo logging pass the `--v`
* To run with klog verbosity, pass the `--focus="g0" -- -v=3` where 3 is the log level: 1-3
