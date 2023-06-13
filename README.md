# free5gc-operator
Sample free5gc operator for Nephio

## Description
Nephio free5gc operator takes the Nephio community produced XXXDeployment (where XXX = AMF | SMF | UPF) custom resources, and deploys the corresponding free5gc AMF | SMF | UPF onto the cluster based on the CR's specifications.

## Getting Started
Prior to running free5gc operator, multus needs to be installed on cluster, and standard CNI binaries need to be installed on /opt/cni/bin directory. We can verify these conditions via:

```sh
$ kubectl get daemonset -n kube-system
NAME             DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR            AGE
kube-multus-ds   1         1         1       1            1           <none>                   23d

$ kubectl get pods -n kube-system | grep multus
kube-multus-ds-ljs8l                       1/1     Running   0          23d
```

and 

```sh
$ ls -l /opt/cni/bin
total 51516
-rwxr-xr-x 1 root root 3056120 Sep 17 08:14 bandwidth
-rwxr-xr-x 1 root root 3381272 Sep 17 08:14 bridge
-rwxr-xr-x 1 root root 9100088 Sep 17 08:14 dhcp
-rwxr-xr-x 1 root root 4425816 Sep 17 08:14 firewall
-rwxr-xr-x 1 root root 2232440 Sep 17 08:14 flannel
-rwxr-xr-x 1 root root 2990552 Sep 17 08:14 host-device
-rwxr-xr-x 1 root root 2580024 Sep 17 08:14 host-local
-rwxr-xr-x 1 root root 3138008 Sep 17 08:14 ipvlan
-rwxr-xr-x 1 root root 2322808 Sep 17 08:14 loopback
-rwxr-xr-x 1 root root 3187384 Sep 17 08:14 macvlan
-rwxr-xr-x 1 root root 2859000 Sep 17 08:14 portmap
-rwxr-xr-x 1 root root 3332088 Sep 17 08:14 ptp
-rwxr-xr-x 1 root root 2453976 Sep 17 08:14 sbr
-rwxr-xr-x 1 root root 2092504 Sep 17 08:14 static
-rwxr-xr-x 1 root root 2421240 Sep 17 08:14 tuning
-rwxr-xr-x 1 root root 3138008 Sep 17 08:14 vlan
```

for free5gc, at least macvlan needs to be installed.

### Loading the CRD
Under the free5gc-operator directory, do:

```sh
make install
```

and the following CRD should be loaded:

```sh
$ kubectl get crds | grep nephio
upfdeployments.workload.nephio.io                         2022-10-10T07:54:28Z
```

### Run the controller

Under the free5gc-operator directory, do:

```sh
make run
```

### Deploy the controller
Change image varible in Makefile:

```sh
REGISTRY ?= registry
PROJECT ?= free5gc-operator
TAG ?= latest
```

build and push free5gc-operator container into registry

```sh
$ make docker-build docker-push
test -s free5gc-operator/bin/controller-gen || GOBIN=free5gc-operator/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.2
free5gc-operator/bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
free5gc-operator/bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."
go fmt ./...
go vet ./...
KUBEBUILDER_ASSETS="io.kubebuilder.envtest/k8s/1.24.2-darwin-arm64" go test ./... -coverprofile cover.out
?   	github.com/nephio-project/free5gc	[no test files]
?   	github.com/nephio-project/free5gc/api/v1alpha1	[no test files]
ok  	github.com/nephio-project/free5gc/controllers	0.567s	coverage: 33.2% of statements
docker build -t registry/free5gc-operator-controller:latest .
[+] Building 31.6s (18/18) FINISHED
 => [internal] load build definition from Dockerfile                                                                                                                                                        0.0s
 => => transferring dockerfile: 37B                                                                                                                                                                         0.0s
 => [internal] load .dockerignore                                                                                                                                                                           0.0s
 => => transferring context: 35B                                                                                                                                                                            0.0s
 => [internal] load metadata for gcr.io/distroless/static:nonroot                                                                                                                                           3.5s
 => [internal] load metadata for docker.io/library/golang:1.19                                                                                                                                              4.6s
 => [auth] library/golang:pull token for registry-1.docker.io                                                                                                                                               0.0s
 => [builder 1/9] FROM docker.io/library/golang:1.19@sha256:d048786fc9274581a1bb7199835b7adee5ae9359fb3501a879cc0d98d8d02d7e                                                                                0.0s
 => [internal] load build context                                                                                                                                                                           0.0s
 => => transferring context: 28.70kB                                                                                                                                                                        0.0s
 => CACHED [stage-1 1/3] FROM gcr.io/distroless/static:nonroot@sha256:149531e38c7e4554d4a6725d7d70593ef9f9881358809463800669ac89f3b0ec                                                                      0.0s
 => CACHED [builder 2/9] WORKDIR /workspace                                                                                                                                                                 0.0s
 => CACHED [builder 3/9] COPY go.mod go.mod                                                                                                                                                                 0.0s
 => CACHED [builder 4/9] COPY go.sum go.sum                                                                                                                                                                 0.0s
 => CACHED [builder 5/9] RUN go mod download                                                                                                                                                                0.0s
 => CACHED [builder 6/9] COPY main.go main.go                                                                                                                                                               0.0s
 => CACHED [builder 7/9] COPY api/ api/                                                                                                                                                                     0.0s
 => [builder 8/9] COPY controllers/ controllers/                                                                                                                                                            0.0s
 => [builder 9/9] RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go                                                                                                                 26.7s
 => [stage-1 2/3] COPY --from=builder /workspace/manager .                                                                                                                                                  0.1s
 => exporting to image                                                                                                                                                                                      0.1s
 => => exporting layers                                                                                                                                                                                     0.1s
 => => writing image sha256:df95b3f11877fd0bec1e04cdb31b5cf03bf474f11b8c6ea1726f5df1218c8c38                                                                                                                0.0s
 => => naming to registry/free5gc-operator-controller:latest                                                                                                                                  0.0s
docker push registry/free5gc-operator-controller:latest
The push refers to repository [registry/free5gc-operator-controller]
47dc1a25876f: Pushed
4cb10dd2545b: Layer already exists
d2d7ec0f6756: Layer already exists
1a73b54f556b: Layer already exists
e624a5370eca: Layer already exists
d52f02c6501c: Layer already exists
ff5700ec5418: Layer already exists
399826b51fcf: Layer already exists
6fbdf253bbc2: Layer already exists
23d0c2981a15: Layer already exists
latest: digest: sha256:3e86a217e906ebfaef66dc7363698e59576c928768105d303d25c59706a01b4d size: 2402
```

and finnaly deploy container to a kubernetes cluster

```sh
$ make deploy
test -s free5gc-operator/bin/controller-gen || GOBIN=free5gc-operator/bin go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.2
free5gc-operator/bin/controller-gen rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
test -s free5gc-operator/bin/kustomize || { curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s -- 3.8.7 free5gc-operator/bin; }
cd config/manager && free5gc-operator/bin/kustomize edit set image controller=registry/free5gc-operator-controller:latest
free5gc-operator/bin/kustomize build config/default | "kubectl" apply -f -
namespace/free5gc-operator-system created
customresourcedefinition.apiextensions.k8s.io/upfdeployments.workload.nephio.org unchanged
serviceaccount/free5gc-operator-controller-manager created
role.rbac.authorization.k8s.io/free5gc-operator-leader-election-role created
clusterrole.rbac.authorization.k8s.io/free5gc-operator-manager-role created
rolebinding.rbac.authorization.k8s.io/free5gc-operator-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/free5gc-operator-manager-rolebinding created
deployment.apps/free5gc-operator-controller-manager created

kubectl get all -n free5gc-operator-system
NAME                                                       READY   STATUS    RESTARTS   AGE
pod/free5gc-operator-controller-manager-59545f64f8-wjngx   2/2     Running   0          92m

NAME                                                  READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/free5gc-operator-controller-manager   1/1     1            1           92m

NAME                                                             DESIRED   CURRENT   READY   AGE
replicaset.apps/free5gc-operator-controller-manager-59545f64f8   1         1         1       92m
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

