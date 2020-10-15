# observability-e2e-test

This is a container which will be called from:
1. Canary Tests
1. Regular Build PRs

This will be called after Observability is installed - both Hub and Addon in real OCP clusters or Kind.

The tests in this container will:
1. Create the MCO CR . The Object store to be already in place for CR to work.
1. Wait for the installation to complete.
1. Then check the entire Observability suite (Hub and Addon) is working as expected including disable/enable, Grafana etc.

Once this is up and running, the e2e tests in the multicluster-monitoring-operator and metrics-collector will be phased out.

## To do a local test using the docker test container:
1. clone this repo
1. copy ./resources/options.yaml.template ./resource/options.yaml , and update values specific to your environment
1. oc login to your cluster in which observability is installed - and make sure that remains the current-context in kubeconfig
1. run `make build`. This will create a docker image. 
1. run `docker images|grep observability-e2e-test` to get the docker-image-id. We will use this in the next step
1. run `docker run --volume /Users/jbanerje/.kube/:/opt/.kube --volume $(pwd)/results:/results --volume $(pwd)/resources:/resources $docker-image-id ` (replace jbanerje with your name!)

In Canary environment, this is the container that will be run - and all the volumes etc will passed on while starting the docker container using a helper script.