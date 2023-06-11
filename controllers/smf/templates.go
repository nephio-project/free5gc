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

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

const configurationTemplateSource = `
info:

  version: 1.0.2
  description: SMF configuration

configuration:

  serviceNameList:
  - nsmf-pdusession
  - nsmf-event-exposure
  - nsmf-oam

  sbi:
    scheme: http
    registerIPv4: smf-nsmf
    bindingIPv4: 0.0.0.0
    port: 80
    tls:
      key: config/TLS/smf.key
      pem: config/TLS/smf.pem

  nrfUri: http://nrf-nnrf:8000
  pfcp:
    addr: {{ .PFCP_IP }}
  smfName: SMF

  snssaiInfos:
  - sNssai:
      sst: 1
      sd: 010203
    dnnInfos:
    - dnn: internet 
      dns:
        ipv4: 8.8.8.8 
  - sNssai:
      sst: 1
      sd: 112233
    dnnInfos:
    - dnn: internet
      dns:
        ipv4: 8.8.8.8
  - sNssai:
      sst: 2
      sd: 112234
    dnnInfos:
    - dnn: internet
      dns:
        ipv4: 8.8.8.8

  plmnList:
  - mcc: "208"
    mnc: "93"

  userplaneInformation:
    upNodes:
      gNB1:
        type: AN
      UPF:
        type: UPF
        nodeID: 10.100.50.241
        sNssaiUpfInfos:
        - sNssai:
            sst: 1
            sd: 010203
          dnnUpfInfoList: {{- range $netInstance := .DNN_LIST }}
  {{- range $dnn := $netInstance.DataNetworks }}
            - dnn: {{ $dnn.Name }}
              pools:
              - cidr: {{(index $dnn.Pool 0).Prefix}}
  {{- end }}
{{- end}}
        interfaces:
        - interfaceType: N3
          endpoints:
          - 10.100.50.233
          networkInstance: internet
    links:
    - A: gNB1
      B: UPF

  locality: area1

logger:
  Aper:
    ReportCaller: false
    debugLevel: info
  NAS:
    ReportCaller: false
    debugLevel: info
  NGAP:
    ReportCaller: false
    debugLevel: info
  PFCP:
    ReportCaller: false
    debugLevel: info
  SMF:
    ReportCaller: false
    debugLevel: debug
`

const ueRoutingConfigurationTemplateSource = `
info:

  version: 1.0.1
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
`

var (
	configurationTemplate          = template.Must(template.New("SMFConfiguration").Parse(configurationTemplateSource))
	ueRoutingConfigurationTemplate = template.Must(template.New("SMFUERoutingConfiguration").Parse(ueRoutingConfigurationTemplateSource))
)

type configurationTemplateValues struct {
	PFCP_IP  string
	DNN_LIST []nephiov1alpha1.NetworkInstance
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
