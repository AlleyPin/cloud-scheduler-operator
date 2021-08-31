# Cloud-Scheduler-Operator
[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg?color=success)](http://www.apache.org/licenses/LICENSE-2.0)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/alleypin/cloud-scheduler-operator?color=69D7E5)
## Introduction
Cloud Scheduler Operator is a [Kubernetes operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) that developed by [Kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) that can be used to set up and configurate a fully cloud managed cron job service to execute specified tasks at regular intervals. For the feature details please check [Cloud Scheduler](https://cloud.google.com/scheduler)


WARNING: Cloud Scheduler Operator will only manage Scheduler resources created by itself in Kubernetes, and when removing resources, it will not delete entities on the cloud

## Usage

First, you must [install the operators](https://book.kubebuilder.io/quick-start.html#run-it-on-the-cluster). After installation is complete, create a [service account](https://cloud.google.com/iam/docs/service-accounts) and binding a role `Cloud Scheduler Admin`, download the json certificate, and store the certificate as Kubernetes secret resource. Finally, create a Scheduler YAML specification by following one of the samples, like [samples/gcp-contrib_v1beta1_scheduler.yaml](./samples/gcp-contrib_v1beta1_scheduler.yaml). 

NOTICE: The sample will create a cloud scheduler regular publish a task to Cloud Pub/Sub topic.


## License
This project is distributed under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0), see [LICENSE](./LICENSE) for more information.
