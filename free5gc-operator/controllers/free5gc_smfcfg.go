/*

 */

package controllers

var SMFCfgTemplate string = `
info:
  version: 1.0.0
  description: UPF configuration

configuration:
  ReportCaller: false
  debugLevel: info
  pfcp:
    - addr: {{ .PFCP_IP }}
  n4:
    - addr: {{ .N4_IP }}
  n11:
    - addr: {{ .N11_IP }}
`
