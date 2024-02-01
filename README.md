free5gc-operator
================

A Kubernetes operator for [free5GC](https://free5gc.org/).

Description
-----------

Manages deployments of free5GC's AMF, SMF, and UPF network functions by reconciling Nephio's
`NFDeployment` custom resources for the `amf.free5gc.io`, `smf.free5gc.io`, and `upf.free5gc.io`
providers.

Getting Started
---------------

### Deploy the CRDs

We need the Nephio API CRDs from the [api repository](https://github.com/nephio-project/api):

```sh
TAG=main
kubectl apply -f https://raw.githubusercontent.com/nephio-project/api/$TAG/config/crd/bases/workload.nephio.org_nfdeployments.yaml
kubectl apply -f https://raw.githubusercontent.com/nephio-project/api/$TAG/config/crd/bases/workload.nephio.org_nfconfigs.yaml
kubectl apply -f https://raw.githubusercontent.com/nephio-project/api/$TAG/config/crd/bases/ref.nephio.org_configs.yaml
```

(Replace `TAG` with a specific tagged version, e.g. `v2.0.0`)

### Run the Operator

Multus needs to be installed on cluster with the "macvlan" CNI plugin.

For testing, you can run the operator locally against the cluster:

```sh
make run
```

Or you can build an image:

```sh
make docker-build docker-push REGISTRY=myregistry
```

(Use your own Docker Hub registry)

Then deploy it the cluster:

```sh
make deploy REGISTRY=myregistry
```

### Deploy Test CRs

```sh
kubectl apply -f test/
```

License
-------

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

