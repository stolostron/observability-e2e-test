# observability-e2e-test

This is modeled after: https://github.com/open-cluster-management/open-cluster-management-e2e

This is a container which will be called from:

1. Canary Tests
2. Regular Build PRs

This will be called after Observability is installed - both Hub and Addon in real OCP clusters or Kind.

The tests in this container will:

1. Create the MCO CR. The Object store is in place for CR to work.
2. Wait for the installation to complete.
3. Then check the entire Observability suite (Hub and Addon) is working as expected including disable/enable, Grafana etc.

## To do a local test using the docker test container:

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

4. oc login to your cluster in which observability is installed - and make sure that remains the current-context in kubeconfig

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

7. run testing

```
$ docker run -v ~/.kube/:/opt/.kube -v $(pwd)/results:/results -v $(pwd)/resources:/resources --env-file $(pwd)/resources/env.list  $docker_image_id
```

In Canary environment, this is the container that will be run - and all the volumes etc will passed on while starting the docker container using a helper script.
