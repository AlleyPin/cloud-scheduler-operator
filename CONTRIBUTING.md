# Contributing Guidelines

Thank you for your interest in contributing to our project. Whether it's a bug report, new feature, correction, or additional
documentation, we greatly value feedback and contributions from our community.

Please read through this document before submitting any issues or pull requests to ensure we have all the necessary
information to effectively respond to your bug report or contribution.

## Getting Started
### Run locally
This guide assumes that you have a k8s cluster setup and can access it via kubectl.
Make sure your k8s cluster server version is >=1.12 and client version >=1.15

To register the CRD in the cluster and create installers:
```
make install
```

To run the controller, run the following command. The controller runs in an infinite loop so open another terminal to create CRDs.
```
make run 
```

To create a Scheduler via the operator, apply a job config:
```
kubectl apply -f samples/gcp-contrib_v1beta1_scheduler.yaml
```

### Deploying to a cluster

You must first push a Docker image containing the changes to a Docker repository like GCR.

### Build and push docker image to GCR

```bash
$ make docker-build docker-push IMG=gcr.io/<YOUR PROJECT ID>/<GCR REPOSITORY>
```

#### Deployment

You must specify GCP service account credentials for the operator. You can do so via environment variables, or by creating them.

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: service-account-secret
type: Opaque
data:
  sa-key.json: <BASE64 encode of certificate json>
EOF
```

#### Verify installation

Run the following to verify that the CRD was installed:
```bash
$ kubectl get crd | grep schedulers
schedulers.gcp-contrib.alleypinapis.com               2021-08-27T11:43:11Z
```

Run the following to verify that the controller container is running:
```bash
$ kubectl get pods -n cloud-scheduler-operator-system
NAME                                                         READY   STATUS    RESTARTS   AGE
cloud-scheduler-operator-controller-manager-678d6897f4-cfk98   2/2     Running   0          50s
```

#### Uninstallation
To remove the operator from your cluster, run:

```bash
$ make undeploy
```

If you created an Service Account for the operator, you can delete it manually in the CCP console.

### Generation of APIs and controllers
The CRD structs, controllers, and other files in this repo were initially created using Kubebuilder:

```bash
go mod init <module_name> (e.g. go mod init Scheduler)
kubebuilder init --domain alleypinapis.com
kubebuilder create api --group gcp-contrib --version v1 --kind Scheduler
```

If you’re not in `GOPATH`, you’ll need to run `go mod init <modulename>` in order to tell kubebuilder and Go the base import path of your module.

For a further understanding, check [installation page](https://book.kubebuilder.io/quick-start.html#installation)

Go package issues:
- Ensure that you activate the module support by running `export GO111MODULE=on` to solve issues such as `cannot find package ... (from $GOROOT).`

## Reporting Bugs/Feature Requests

We welcome you to use the GitHub issue tracker to report bugs or suggest features.

When filing an issue, please check existing open, or recently closed, issues to make sure somebody else hasn't already
reported the issue. Please try to include as much information as you can. Details like these are incredibly useful:

* A reproducible test case or series of steps
* The version of our code being used
* Any modifications you've made relevant to the bug
* Anything unusual about your environment or deployment


## Contributing via Pull Requests
Contributions via pull requests are much appreciated. Before sending us a pull request, please ensure that:

1. You are working against the latest source on the *master* branch.
2. You check existing open, and recently merged, pull requests to make sure someone else hasn't addressed the problem already.
3. You open an issue to discuss any significant work - we would hate for your time to be wasted.

To send us a pull request, please:

1. Fork the repository.
2. Modify the source; please focus on the specific change you are contributing. If you also reformat all the code, it will be hard for us to focus on your change.
3. Ensure local tests pass.
4. Commit to your fork using clear commit messages.
5. Send us a pull request, answering any default questions in the pull request interface.
6. Pay attention to any automated CI failures reported in the pull request, and stay involved in the conversation.

GitHub provides additional document on [forking a repository](https://help.github.com/articles/fork-a-repo/) and
[creating a pull request](https://help.github.com/articles/creating-a-pull-request/).


## Finding contributions to work on
Looking at the existing issues is a great way to find something to contribute on. As our projects, by default, use the default GitHub issue labels (enhancement/bug/duplicate/help wanted/invalid/question/wontfix), looking at any 'help wanted' issues is a great place to start.



## Licensing
See the [LICENSE](LICENSE) file for our project's licensing.
