free5gc-operator
================

free5GC operator for Nephio.

Description
-----------

The Nephio free5GC operator takes the Nephio community produced XXXDeployment (where XXX = AMF | SMF | UPF) custom resources, and deploys the corresponding free5GC AMF | SMF | UPF onto the cluster based on the CR's specifications.

Getting Started
---------------

Prior to running the free5GC operator, Multus needs to be installed on cluster with the macvlan CNI.

### Deploy the CRDs

```sh
make install
```

### Run the Operator

For testing, you can run the operator locally against the cluster:

```sh
make run
```

### Deploy the Operator

Use your own Docker Hub registry:

```sh
make docker-build docker-push REGISTRY=myregistry
```

Then deploy to the cluster:

```sh
make deploy REGISTRY=myregistry
```

## License

Copyright 2023 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

