/*
  Free5GC Config
  N4:  NF Deployment IF

  sbi:
        scheme: http
        registerIPv4: smf-nsmf # IP used to register to NRF
        bindingIPv4: 0.0.0.0  # IP used to bind the service
        port: 80
        tls:
          key: config/TLS/smf.key
          pem: config/TLS/smf.pem

      nrfUri: http://nrf-nnrf:8000

      pfcp:
        addr: 10.100.50.244
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
`
