# Contribution Guide

## Building

### Golang

This project assumes you have golang 1.17 or higher installed on your machine.

### Container Image

This project uses [ko](https://github.com/google/ko) to build the container image with the
controller manager and deploy it to a Kubernetes cluster. To change the destination container
registry, provide the `IMAGE_REPO` variable to any make target:

```sh
$ make container-build IMAGE_REPO=quay.io/myusername
```

Similarly, use the `TAG` variable to modify the tag for the built image.
By default, this repository builds and pushes the controller image to
`ghcr.io/adambkaplan/kubebuilder-plus/project:latest`.

ko by default generates a Software Bill of Materials (SBOM) alongside the container image, which
can be pushed to a container registry.
Not all container registries support SBOM images - to disable this, set the `SBOM` variable to
`none`:

```sh
$ make container-push IMAGE_REPO=quay.io/myusername SBOM=none
```

If you need finer control over how ko builds and pushes images, provide all necessary ko
arguments to the `KO_OPTS` variable:

```sh
$ make container-build IMAGE_REPO=quay.io/myusername/kubebuilder-plus KO_OPTS="--bare --tag=mytag"
```

By default the following arguments are passed to ko:

- `-B`: this strips the md5 hash from the image name
- `-t ${TAG}`: sets the tag for the image. Defaults to the `TAG` argument (`latest`)
- `--sbom=${SBOM}`: configures SBOM behavior. Defaults to the `SBOM` argument (`spdx`)

## Testing

- Ensure unit tests pass by running `make test`.
- Ensure integration tests pass by running `make test-integration`.
- Ensure that [cert-manager](https://cert-manager.io) has been installed on your cluster.
- Ensure that you are able to build and deploy the controller image to your Kubernetes cluster:

   ```sh
   $ make deploy IMAGE_REPO=quay.io/myusername
   ```

   *Note*: This will publish the container image to `quay.io/myusername/project:latest`. For the
   deployment to succeed, your cluster must have permission to pull this image.

- Optionally, ensure end to end tests succeed by running `make test-e2e`.

### KinD

TODO: Provide instructions on how to deploy with KinD

### CodeReady Containers

To deploy on an OpenShift CodeReady Containers cluster, do the following:

0. Install the cert-manager operator from OperatorHub.

1. Create a project for the deployment (project-system by default)

   ```sh
   $ oc new-project project-system
   ```

2. Use the following options for `make deploy`:

   ```sh
   $ make deploy IMAGE_REPO=default-route-openshift-image-registry.apps-crc.testing/project-system KO_OPTS="-B -t latest --sbom=none --insecure-registry"
   ```
