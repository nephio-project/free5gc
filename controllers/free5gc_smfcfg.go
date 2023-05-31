package controllers

var SMFCfgTemplate string = `
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
            ipv4: {{ .DNS_IP }}
    - sNssai:
        sst: 1
        sd: 112233
      dnnInfos:
        - dnn: internet
          dns:
            ipv4: {{ .DNS_IP }}
    - sNssai:
        sst: 2
        sd: 112234
      dnnInfos:
        - dnn: internet
          dns:
            ipv4: {{ .DNS_IP }}
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
            dnnUpfInfoList:
              - dnn: {{ $dnn.Name }}
                pools:
                  - cidr: {{(index $dnn.Pool 0).Prefix}}
          - sNssai:
              sst: 1
              sd: 112233
            dnnUpfInfoList:
              - dnn: {{ $dnn.Name }}
                pools:
                  - cidr: {{(index $dnn.Pool 0).Prefix}}
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
