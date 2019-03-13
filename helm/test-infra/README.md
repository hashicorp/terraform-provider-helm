# Testing Infrastructure

Testing the Helm provider should not require more than a working Kubernetes cluster,
so any testing environment from the Kubernetes provider should work.

To keep this provider self-contained we provide code for one of these environments, GKE.
Please follow instructions in the `gke` directory to spin up GKE.

## Helm installation

It is necessary to install Helm after spinning up the GKE cluster.
We provide a shell script which leverages Docker, for convenience.

Here we assume that you chose to store kubeconfig in `./gke/kubedir`,
feel free to change this location accordingly, if necessary.

```sh
KUBE_DIR=./gke/kubedir HELM_VERSION=2.13.0 HELM_HOME=./helm-home HYPERKUBE_VERSION=v1.11.8 ./install-helm-via-docker.sh
```

Then you're ready to run acceptance tests.

```sh
HELM_HOME=./helm-home KUBECONFIG=./gke/kubedir/config make testacc
```
