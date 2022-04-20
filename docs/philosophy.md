# Kubebuilder-plus Philosophy

This outlines the principles and philosophies behind the changes made in this project.

## Foundation: Kubebuilder

Kubebuilder is an upstream Kubernetes project designed to simplify controller development.
The project and it's related libraries (such as controller-runtime) take care of a lot of repeated
and otherwise complex code needed to implement a Kubernetes controller.
The project includes a CLI designed that bootstraps the creation of custom resources, controllers,
and webhooks.

I have chosen Kubebuilder as the basis of the project, using the contents from the CronJob
tutorial.

## Refactor for Testing

The kubebuilder community recommends integration-style testing for controllers, and provides the
envtest utility to run controllers against a simulated Kubernetes cluster.
The community further recommends Ginkgo to structure tests, as this library encourages behavior-
driven test structures.

In practice, this position negatively impacts contributor experience by adding friction to the
"red-green-refactor" cycle that many software practitioners advocate.
Envtest adds noticeable overhead to test setup and execution.
Ginkgo tests also difficult to debug, as many IDEs do not natively support this testing library.
I attempted to reduce this friction by moving integration tests to their own directory tree, with
appropriate updates to the Makefile to run the integration tests separately.
The separate directory structure also lends itself to the creation of end to end (e2e) tests
against real clusters.

I also created a separate `pkg/` directory to hold "business" logic of the controller.
In the original tutorial, this logic is encapsulated in anonymous functions within the main
reconciliation loop.
While useful for a compact tutorial, this approach does not scale as more features are added to
the controller and APIs.
By moving the anonymous functions to separate packages, new unit tests were written.
These tests do not have the overhead of envtest, and can execute in the order of milliseconds.
Using standard golang unit tests also encourages higher quality code, since most go developers are
accustomed to writing tests with the standard library.

## Build and Deploy with ko

Secure software best practices recommend that all software should provide a Software Bill of
Materials (SBOM) itemizing all dependencies.
SBOMs are most accurate if they are generated when the code is compiled and built.
SBOMs are less accurate if the toolchain is not able to detect and categorize dependencies.

ko is an opinionated toolchain that builds container images for go applications.
It generates SBOMs for builds by default, and has other features that improve security (ex -
using distroless images).
ko also has features that simplify the deployment of an application on Kubernetes.
It therefore makes sense to use this as the tool of choice for building the project.
