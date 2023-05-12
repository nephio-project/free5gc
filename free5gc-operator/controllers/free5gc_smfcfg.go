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
  description: SMF configuration

configuration:
  ReportCaller: false
  debugLevel: info
  serviceNameList:
    - nsmf-pdusession
    - nsmf-event-exposure
    - nsmf-oam
      
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
    addr: {{ .PFCP_IP }}
  smfName: SMF
  snssaiInfos:
    - sNssai:
        sst: 1
        sd: 010203
      dnnInfos: # DNN information list
        - dnn: internet # Data Network Name
          dns: # the IP address of DNS
            ipv4: 8.8.8.8
    - sNssai:
        sst: 1
        sd: 112233
      dnnInfos: # DNN information list
        - dnn: internet # Data Network Name
          dns: # the IP address of DNS
            ipv4: 8.8.8.8
    - sNssai:
        sst: 2
        sd: 112234    
      dnnInfos:
        - dnn: internet
          dns: 
            ipv4: 8.8.8.8
  plmnList: # the list of PLMN IDs that this SMF belongs to (optional, remove this key when unnecessary)
    - mcc: "208" # Mobile Country Code (3 digits string, digit: 0~9)
      mnc: "93" # Mobile Network Code (2 or 3 digits string, digit: 0~9)
  userplaneInformation: # list of userplane information
    upNodes: # information of userplane node (AN or UPF)
      gNB1: # the name of the node
        type: AN # the type of the node (AN or UPF)
      UPF:  # the name of the node
        type: UPF # the type of the node (AN or UPF)
        nodeID: 10.100.50.241 # the IP/FQDN of N4 interface on this UPF (PFCP)
        sNssaiUpfInfos: # S-NSSAI information list for this UPF
              - sNssai: # S-NSSAI (Single Network Slice Selection Assistance Information)
                  sst: 1 # Slice/Service Type (uinteger, range: 0~255)
                  sd: 010203 # Slice Differentiator (3 bytes hex string, range: 000000~FFFFFF)
                dnnUpfInfoList: # DNN information list for this S-NSSAI
                  - dnn: internet
                    pools:
                      - cidr: 10.1.0.0/17
              - sNssai: # S-NSSAI (Single Network Slice Selection Assistance Information)
                  sst: 1 # Slice/Service Type (uinteger, range: 0~255)
                  sd: 112233 # Slice Differentiator (3 bytes hex string, range: 000000~FFFFFF)
                dnnUpfInfoList: # DNN information list for this S-NSSAI
                  - dnn: internet
                    pools:
                      - cidr: 10.1.128.0/17 
        interfaces: # Interface list for this UPF
              - interfaceType: N3 # the type of the interface (N3 or N9)
                endpoints: # the IP address of this N3/N9 interface on this UPF
                  - 10.100.50.233
                networkInstance: internet # Data Network Name (DNN)
    links: # the topology graph of userplane, A and B represent the two nodes of each link
      - A: gNB1
        B: UPF
  locality: area1 # Name of the location where a set of AMF, SMF and UPFs are located

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
    debugLevel: info
`
