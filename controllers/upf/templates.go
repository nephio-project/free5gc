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

package upf

import (
	"bytes"
	"text/template"

	nephiov1alpha1 "github.com/nephio-project/api/nf_deployments/v1alpha1"
)

const configurationTemplateSource = `
info:

  version: 1.0.0
  description: UPF configuration

configuration:

  ReportCaller: false
  debugLevel: info
  dnn_list:
{{- range $netInstance := .N6cfg }}
  {{- range $dnn := $netInstance.DataNetworks }}
  - cidr: {{(index $dnn.Pool 0).Prefix}}
    dnn: {{ $dnn.Name }}
    natifname: {{index $netInstance.Interfaces 0}}
  {{- end }}
{{- end}}
  pfcp:
  - addr: {{ .PFCP_IP }}

  gtpu:
  - addr: {{ .GTPU_IP }}
 
  # [optional] gtpu.name
  # - name: upf.5gc.nctu.me
  # [optional] gtpu.ifname
  # - ifname: gtpif
`

const wrapperScriptTemplateSource = `#!/bin/sh

### Implement networking rules
iptables -A FORWARD -j ACCEPT
{{- range $netInstance := .N6cfg }}
  {{- range $dnn := $netInstance.DataNetworks }}
iptables -t nat -A POSTROUTING -s {{(index $dnn.Pool 0).Prefix}} -o {{index $netInstance.Interfaces 0}} -j MASQUERADE  # route traffic comming from the UE  SUBNET to the interface N6
  {{- end }}
{{- end }}
echo "1200 n6if" >> /etc/iproute2/rt_tables # create a routing table for the interface N6
{{- range $netInstance := .N6cfg }}
  {{- range $dnn := $netInstance.DataNetworks }}
ip rule add from {{(index $dnn.Pool 0).Prefix}} table n6if   # use the created ip table to route the traffic comming from  the UE SUBNET
ip route add default via {{$.N6gw}} dev {{index $netInstance.Interfaces 0}} table n6if  # add a default route in the created table so  that all UEs will use this gateway for external communications (target IP not in the Data Network attached  to the interface N6) and then the Data Network will manage to route the traffic
  {{- end }}
{{- end }}

/free5gc/free5gc-upfd/free5gc-upfd -c /free5gc/config//upfcfg.yaml
`

var (
	configurationTemplate = template.Must(template.New("UPFConfiguration").Parse(configurationTemplateSource))
	wrapperScriptTemplate = template.Must(template.New("UPFWrapperScript").Parse(wrapperScriptTemplateSource))
)

type configurationTemplateValues struct {
	PFCP_IP string
	GTPU_IP string
	N6cfg   []nephiov1alpha1.NetworkInstance
	N6gw    string
}

func renderConfigurationTemplate(values configurationTemplateValues) (string, error) {
	var buffer bytes.Buffer
	if err := configurationTemplate.Execute(&buffer, values); err == nil {
		return buffer.String(), nil
	} else {
		return "", err
	}
}

func renderWrapperScriptTemplate(values configurationTemplateValues) (string, error) {
	var buffer bytes.Buffer
	if err := wrapperScriptTemplate.Execute(&buffer, values); err == nil {
		return buffer.String(), nil
	} else {
		return "", err
	}
}
