# cluster-api-provider-ionoscloud
Kubernetes-native declarative infrastructure for [IONOS Cloud](ionos_cloud).

## What is the Cluster API Provider IONOS Cloud

The [Cluster API][cluster_api] brings declarative, Kubernetes-style APIs to cluster creation, configuration and management. Cluster API Provider for IONOS Cloud is a concrete implementation of Cluster API for IONOS Cloud.

The API itself is shared across multiple cloud providers allowing for IONOS Cloud deployments of Kubernetes. It is built atop the lessons learned from previous cluster managers such as [kops][kops] and [kubicorn][kubicorn].

## Launching a Kubernetes cluster on IONOS Cloud

...

## Features

* Native Kubernetes manifests and API
* Manages the bootstrapping of Servers on cluster.
* Choice of Linux distribution between Ubuntu 22.04 and other cloud init distribution using Server Templates based on raw images from [image builder](image_builder).
* Using cloud init for bootstrapping nodes.
* Installs only the minimal components to bootstrap a control plane and workers.

# Roadmap

* full ipv6 cluster
* dual stack mode (ipv4 and ipv6)
* private cluster
* managed kubernetes
* failuredomains for control planes
* failuredomains for machinedeployment
* autoscaler integrations example
* multi lan (private and public lan)

---

## Compatibility with Cluster API

|                       | Cluster API v1beta1 (v1.5) |
| :-------------------: | :------------------------: |
| CAPIC v1alpha1 (v0.1) |             ✓              |



## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

<!-- References -->
[cluster_api]: https://github.com/kubernetes-sigs/cluster-api
[kops]: https://github.com/kubernetes/kops
[kubicorn]: http://kubicorn.io/
[image_builder]: https://github.com/kubernetes-sigs/image-builder/
[ionos_cloud]: https://ionos.cloud