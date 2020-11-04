# observability-e2e-test

This is modeled after: https://github.com/open-cluster-management/open-cluster-management-e2e

This is a container which will be called from:

1. Canary Tests
2. Regular Build PRs

This will be called after Observability is installed - both Hub and Addon in real OCP clusters or Kind.

The tests in this container will:

1. Create the MCO CR . The Object store to be already in place for CR to work.
2. Wait for the installation to complete.
3. Then check the entire Observability suite (Hub and Addon) is working as expected including disable/enable, Grafana etc.

## Running E2E

1. clone this repo:

```
$ git clone git@github.com:open-cluster-management/observability-e2e-test.git
```

2. copy `resources/options.yaml.template` to `resources/options.yaml`, and update values specific to your environment:

```
$ cp resources/options.yaml.template resources/options.yaml
$ cat resources/options.yaml
options:
  hub:
    baseDomain: BASE_DOMAIN
    user: BASE_USER
    password: BASE_PASSWORD
```

3. run testing:

```
$ export BUCKET=YOUR_S3_BUCKET
$ export REGION=YOUR_S3_REGION
$ export AWS_ACCESS_KEY_ID=YOUR_S3_AWS_ACCESS_KEY_ID
$ export AWS_SECRET_ACCESS_KEY=YOUR_S3_AWS_SECRET_ACCESS_KEY
$ export KUBECONFIG=~/.kube/config
$ ginkgo -v -- -options=resources/options.yaml
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
  hub:
    baseDomain: BASE_DOMAIN
    user: BASE_USER
    password: BASE_PASSWORD
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

5. run `make build`. This will create a docker image:

```
$ make build
```

6. run the following command to get docker image ID, we will use this in the next step:

```
$ docker_image_id=`docker images | grep observability-e2e-test | sed -n '1p' | awk '{print $3}'`
```

7. run testing:

```
$ docker run -v ~/.kube/:/opt/.kube -v $(pwd)/results:/results -v $(pwd)/resources:/resources --env-file $(pwd)/resources/env.list  $docker_image_id
```

In Canary environment, this is the container that will be run - and all the volumes etc will passed on while starting the docker container using a helper script.

## Contributing to E2E

### Options.yaml

The values in the options.yaml are optional values read in by E2E. If you do not set an option, the test case that depends on the option should skip the test. The sample values in the option.yaml.template should provide enough context for you fill in with the appropriate values. Further, in the section below, each test should document their test with some detail.

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
