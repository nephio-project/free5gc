/*
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
*/

package smf

import (
	"bytes"
	"text/template"

	nephiov1alpha1 "github.com/nephio-project/api/workload/v1alpha1"
)

const configurationTemplateSource = `
info:

  version: 1.0.7
  description: SMF configuration

configuration:

  serviceNameList:
  - nsmf-pdusession
  - nsmf-event-exposure
  - nsmf-oam

  sbi:
    scheme: http
    registerIPv4: {{ .SVC_NAME }}
    bindingIPv4: 0.0.0.0
    port: 80
    tls:
      key: cert/smf.key
      pem: cert/smf.pem

  nrfUri: http://nrf-nnrf:8000
  nrfCertPem: cert/nrf.pem
  pfcp:
    nodeID: {{ .PFCP_IP }}
    listenAddr: {{ .PFCP_IP }}
    externalAddr: {{ .PFCP_IP }}
    associateFailAlertInterval: 10s
    associateFailRetryInterval: 30s
    heartbeatInterval: 10s
  smfName: SMF

  snssaiInfos:
  - sNssai:
      sst: 1
      sd: 010203
    dnnInfos:
    - dnn: internet
      dns:
        ipv4: 8.8.8.8
        ipv6: 2001:4860:4860::8888
  - sNssai:
      sst: 1
      sd: 112233
    dnnInfos:
    - dnn: internet
      dns:
        ipv4: 8.8.8.8
        ipv6: 2001:4860:4860::8888
  - sNssai:
      sst: 2
      sd: 112234
    dnnInfos:
    - dnn: internet
      dns:
        ipv4: 8.8.8.8
        ipv6: 2001:4860:4860::8888
  plmnList:
  - mcc: "208"
    mnc: "93"
  userplaneInformation:
    upNodes:
{{- range $index, $upf := .UPF_LIST }}
      gNB{{ $index }}:
        type: AN
      {{ $upf.Name }}:
        type: UPF
        nodeID: {{ $upf.N4IP }}
        addr: {{ $upf.N4IP }}
        sNssaiUpfInfos:
        - sNssai:
            sst: 1
            sd: 010203
          dnnUpfInfoList:
  {{- range $n6Instances := $upf.N6Cfg }}
    {{- range $dnn := $n6Instances.DataNetworks }}
            - dnn: {{ $dnn.Name }}
              pools:
              - cidr: {{(index $dnn.Pool 0).Prefix}}
    {{- end }}
  {{- end }}
        - sNssai:
            sst: 1
            sd: 112233
          dnnUpfInfoList:
  {{- range $n6Instances := $upf.N6Cfg }}
    {{- range $dnn := $n6Instances.DataNetworks }}
            - dnn: {{ $dnn.Name }}
              pools:
              - cidr: {{(index $dnn.Pool 1).Prefix}}
    {{- end }}
  {{- end }}
        - sNssai:
            sst: 2
            sd: 112234
          dnnUpfInfoList:
  {{- range $n6Instances := $upf.N6Cfg }}
    {{- range $dnn := $n6Instances.DataNetworks }}
            - dnn: {{ $dnn.Name }}
              pools:
              - cidr: {{(index $dnn.Pool 2).Prefix}}
    {{- end }}
  {{- end }}
        interfaces:
        - interfaceType: N3
          endpoints:
          - {{ $upf.N3IP }}
          networkInstances:
          - internet
{{- end}}
    links:
{{- range $index, $upf := .UPF_LIST }}
    - A: gNB{{ $index }}
      B: {{ $upf.Name }}
{{- end}}

  locality: area1
  t3591:
    enable: true # true or false
    expireTime: 16s # default is 6 seconds
    maxRetryTimes: 3 # the max number of retransmission

  # retransmission timer for PDU session release command
  t3592:
    enable: true # true or false
    expireTime: 16s # default is 6 seconds
    maxRetryTimes: 3 # the max number of retransmission
  urrPeriod: 30 # default usage report period in seconds
  urrThreshold: 500000 # default usage report threshold in bytes
  requestedUnit: 1000

  # Metrics configuration
  # If using the same bindingIPv4 as the sbi server, make sure that the ports are different
  metrics:
    enable: false # (Optional, default false)
    scheme: http # (Required) the protocol for metrics (http or https, default https)
    bindingIPv4: {{ .SVC_NAME }} # (Required) IP used to bind the metrics endpoint (default 0.0.0.0)
    port: 9091 # (Optional, default 9091) port used to bind the service
    tls: # (Optional) the local path of TLS key (Could be the same as the sbi ones)
      pem: cert/smf.pem # SMF TLS Certificate
      key: cert/smf.key # SMF TLS Private key
    namespace: free5gc # (Optional, default free5gc)

logger:
  enable: true # true or false
  level: info # how detailed to output, value: trace, debug, info, warn, error, fatal, panic
  reportCaller: false
`

const ueRoutingConfigurationTemplateSource = `
info:

  version: 1.0.7
  description: Routing information for UE

ueRoutingInfo:

  UE1:
    members:
    - imsi-208930000000003
    topology:
    - A: gNB1
      B: BranchingUPF
    - A: BranchingUPF
      B: AnchorUPF1
    specificPath:
    - dest: 10.100.100.26/32
      path: [BranchingUPF, AnchorUPF2]

  UE2:
    members:
    - imsi-208930000000004
    topology:
    - A: gNB1
      B: BranchingUPF
    - A: BranchingUPF
      B: AnchorUPF1
    specificPath:
    - dest: 10.100.100.16/32
      path: [BranchingUPF, AnchorUPF2]

routeProfile: # Maintains the mapping between RouteProfileID and ForwardingPolicyID of UPF
  MEC1: # Route Profile identifier
    forwardingPolicyID: 10 # Forwarding Policy ID of the route profile

pfdDataForApp: # PFDs for an Application
  - applicationId: edge # Application identifier
    pfds: # PFDs for the Application
      - pfdID: pfd1 # PFD identifier
        flowDescriptions: # Represents a 3-tuple with protocol, server ip and server port for UL/DL application traffic
          - permit out ip from 10.100.100.1 80 to any
`

var (
	configurationTemplate          = template.Must(template.New("SMFConfiguration").Parse(configurationTemplateSource))
	ueRoutingConfigurationTemplate = template.Must(template.New("SMFUERoutingConfiguration").Parse(ueRoutingConfigurationTemplateSource))
)

type UpfPeerConfigTemplate struct {
	Name  string
	N3IP  string
	N4IP  string
	N6Cfg []nephiov1alpha1.NetworkInstance
}

type configurationTemplateValues struct {
	SVC_NAME string
	PFCP_IP  string
	UPF_LIST []UpfPeerConfigTemplate
}

func renderConfigurationTemplate(values configurationTemplateValues) (string, error) {
	var buffer bytes.Buffer
	if err := configurationTemplate.Execute(&buffer, values); err == nil {
		return buffer.String(), nil
	} else {
		return "", err
	}
}

func renderUeRoutingConfigurationTemplate(values configurationTemplateValues) (string, error) {
	var buffer bytes.Buffer
	if err := ueRoutingConfigurationTemplate.Execute(&buffer, values); err == nil {
		return buffer.String(), nil
	} else {
		return "", err
	}
}
