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