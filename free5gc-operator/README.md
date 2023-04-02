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
upfdeploys.nfdeploy.nephio.io                         2022-10-10T07:54:28Z
```

### Run the controller

Under the free5gc-operator directory, do:

```sh
make run
```

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

