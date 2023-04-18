# free5gc operator with AMF/SMF/UPFDeployment CRs for Nephio Release 1
For R1, Nephio free5gc operator primarily performs the following functions:
1. watches for XXXDeployment custom resource (where XXX = {AMF | SMF | UPF}), then constructs and applies corresponding ConfigMap, Deployment, and Service (for AMF and SMF) resources
1. updates XXXDeployment status subresources

This writeup primarily focuses on the first bullet above, with an emphasis on where in the current **NFDeployment** structure can be used for corresponding values in the ConfigMap, Deployment=, and Service specification for the target NF.

## AMFDeployment
[AMFDeployment](https://github.com/nephio-project/api/blob/d42078aea83e769d581f17c8477266d8e1ba41b2/nf_deployments/v1alpha1/amf_deployment_types.go#L27) primarily consists of the [NFDeployment](https://github.com/nephio-project/api/blob/d42078aea83e769d581f17c8477266d8e1ba41b2/nf_deployments/v1alpha1/nf_deployment_types.go#L25) structure. For free5gc AMF, the manifests can be found [here](./sample-manifests/amf.yaml).

**ConfigMap** for AMF only has a single entry: amfcfg.yaml, the following table maps the fields from amfcfg to NFDeployment:

| NFDeployment field | amfcfg.yaml field |
| :----------------- | :----------------- |
| IPv4Config.Address in Interface.Name == "N2" | *ngapIPList* (one entry) |
| IPv4Config.Address in Interface.Name == "Service" | *sbi.registerIPv4* and *bindingIPv4* |
| **missing field** | sbi.port | 
| NetworkInstance.Peers[Name == "NRF"].IPv4Config.Address | address for *nrfUri* |
| **missing info** | *plmnSupportList.mcc/mnc* |
| NetworkInstance.DataNetworks[0].Name | *supportDnnList* (one entry) |

Hard code everything else in amfcfg.yaml

**Service** for AMF has the following mapping between NFDeployment and AMF's Service:

| NFDeployment field | AMF Service field |
| :----------------- | :----------------- |
| **missing info** (hardcode to 80 for now)  | *spec.port* |
| **missing info** (hardcode to 80 for now)  | *spec.targetPort* |

http and TCP should be the default, and aren't expected to change for R1. *ClusterIP* is not correct, and depending on test-infra, it should be *LoadBalancer* (properly should be an input parameter).

**Deployment** for AMF has the following mapping between NFDeployment and AMF's Deployment:

| NFDeployment field | AMF Deployment field |
| :----------------- | :----------------- |
| **from NAD for N2 from package** | *annotations:...networks:name* |
| NetworkInstance.Peers[Name == "NRF"].IPv4Config.Address | *nrf-nnrf* | 

## SMFDeployment
[SMFDeployment](https://github.com/nephio-project/api/blob/d42078aea83e769d581f17c8477266d8e1ba41b2/nf_deployments/v1alpha1/smf_deployment_types.go#L45) primarily consists of the [NFDeployment](https://github.com/nephio-project/api/blob/d42078aea83e769d581f17c8477266d8e1ba41b2/nf_deployments/v1alpha1/nf_deployment_types.go#L25) structure. For free5gc SMF, the manifests can be found [here](./sample-manifests/smf.yaml).

**ConfigMap** for SMF has a two entries: smfcfg.yaml and uerouting.yaml, the following table maps the fields from smfcfg to NFDeployment:

| NFDeployment field | smfcfg.yaml field |
| :----------------- | :----------------- |
| Interfaces[Name == "Service"].IPv4Config.Address | *sbi.registerIPv4* and *bindingIPv4* |
| **missing field** | sbi.port | 
| NetworkInstance.Peers[Name == "NRF"].IPv4Config.Address | address for *nrfUri.nrf-nnrf* |
| Interfaces[Name == "N4"].IPv4Config.Address | *pfcp.addr* (one entry) |
| **missing info** | *snssaiInfos...sst/sd* |
| NetworkInstance.DataNetworks[0].Name | *dnnInfos.dnn* (one entry) |
| **missing info** | *plmnList.mcc/mnc* |
| NetworkInstance.Peers[Name == "UPF's N4?" **need to know the peer name**].IPV4Config.Address | *nodeID* |
| NetworkInstance.DataNetworks[Name == "internet"].UEIPAddressPool | *dnnUpfInfoList.pools.cidr* |
| NetworkInstance.Peers[Name == "UPF's N3?"].IPV4Config.Address | *interfaceType:N3.endpoints* |

Hard code everything else in smfcfg.yaml

from uerouting to NFDeployment:
| NFDeployment field | uerouting field |
| :----------------- | :----------------- |
| NetworkInstance.Peers[Name == "UPF's N6?"IPV4Config.Address; UERANSIM match | *specificPath.dest* |

**Service** for SMF has the following mapping between NFDeployment and SMF's Service:

| NFDeployment field | SMF Service field |
| :----------------- | :----------------- |
| **missing info** (hardcode to 80 for now)  | *spec.port* |
| **missing info** (hardcode to 80 for now)  | *spec.targetPort* |

http and TCP should be the default, and aren't expected to change for R1. *ClusterIP* is not correct, and depending on test-infra, it should be *LoadBalancer* (properly should be an input parameter).

**Deployment** for SMF has the following mapping between NFDeployment and SMF's Deployment:

| NFDeployment field | SMF Deployment field |
| :----------------- | :----------------- |
| **from NAD for N4 from package** | *annotations:...networks:name* |
| NetworkInstance.Peers[Name == "NRF"].IPv4Config.Address | *nrf-nnrf* | 

## UPFDeployment
[UPFDeployment](https://github.com/nephio-project/api/blob/d42078aea83e769d581f17c8477266d8e1ba41b2/nf_deployments/v1alpha1/upf_deployment_types.go#L45) primarily consists of the [NFDeployment](https://github.com/nephio-project/api/blob/d42078aea83e769d581f17c8477266d8e1ba41b2/nf_deployments/v1alpha1/nf_deployment_types.go#L25) structure. For free5gc SMF, the manifests can be found [here](./sample-manifests/upf.yaml).

**ConfigMap** for UPF has a two entries: upfcfg.yaml and wrapper.sh, the following table maps the fields from upfcfg to NFDeployment:

| NFDeployment field | upfcfg.yaml field |
| :----------------- | :----------------- |
| Interfaces[Name == "N4"].IPv4Config.Address | *pfcp.addr / nodeID* |
| Interfaces[Name == "N3"].IPv4Config.Address | *gtpu.iflist.addr* (one entry) |
| NetworkInstance.DataNetworks[0].Name | *dnnList.dnn* (one entry) |
| NetworkInstance.DataNetworks[0].UEIPAddressPool | *dnnList.cidr* (one entry) |

for wrapper.sh:

| NFDeployment field | wrapper.sh field |
| :----------------- | :----------------- |
| NetworkInstance.DataNetworks[0].UEIPAddressPool (bump up CIDR) | *...POSTROUTING -s <cidr>* |
| gateway for Interfaces[Name == "N6"].IPv4Config.Address | *ip route add default via <ip>* |

**Deployment** for UPF has the following mapping between NFDeployment and UPF's Deployment:

| NFDeployment field | SMF Deployment field |
| :----------------- | :----------------- |
| **from NAD for N3 from package** | *annotations:...networks:name* |
| **from NAD for N4 from package** | *annotations:...networks:name* |
| **from NAD for N6 from package** | *annotations:...networks:name* |
