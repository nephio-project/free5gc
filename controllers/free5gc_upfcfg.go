/*

 */

package controllers

var UPFCfgTemplate string = `
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
