/*
  Free5GC Config
  N2:  NF Deployment IF

  sbi:
        scheme: http
        registerIPv4: amf-namf 
        bindingIPv4: 0.0.0.0  
        port: 80
        tls:
          key: config/TLS/smf.key
          pem: config/TLS/smf.pem

      nrfUri: http://nrf-nnrf:8000
*/
package controllers

var AMFCfgTemplate string = `
info:
  version: 1.0.0
  description: AMF configuration

configuration:
  ReportCaller: false
  debugLevel: info
  serviceNameList:
    - namf-comm
    - namf-evts
    - namf-oam
	- namf-mt
	- namf-loc
      
  sbi:
    scheme: http
    registerIPv4: amf-namf 
    bindingIPv4: 0.0.0.0  
    port: 80
    tls:
      key: config/TLS/smf.key
      pem: config/TLS/smf.pem
      
  nrfUri: http://nrf-nnrf:8000

        servedGuamiList:
        - plmnId:
            mcc: 208
            mnc: 93
          amfId: cafe00
      supportTaiList:
        - plmnId:
            mcc: 208
            mnc: 93
          tac: 1
      plmnSupportList:
        - plmnId:
            mcc: 208
            mnc: 93
          snssaiList:
            - sst: 1
              sd: 010203
            - sst: 1
              sd: 112233
      supportDnnList:
        - internet
      security:
        integrityOrder:
          - NIA2
        cipheringOrder:
          - NEA0
      networkName:
        full: free5GC
        short: free      
      locality: area1 # Name of the location where a set of AMF, SMF and UPFs are located
      networkFeatureSupport5GS: # 5gs Network Feature Support IE, refer to TS 24.501
        enable: true # append this IE in Registration accept or not
        length: 1 # IE content length (uinteger, range: 1~3)
        imsVoPS: 0 # IMS voice over PS session indicator (uinteger, range: 0~1)
        emc: 0 # Emergency service support indicator for 3GPP access (uinteger, range: 0~3)
        emf: 0 # Emergency service fallback indicator for 3GPP access (uinteger, range: 0~3)
        iwkN26: 0 # Interworking without N26 interface indicator (uinteger, range: 0~1)
        mpsi: 0 # MPS indicator (uinteger, range: 0~1)
        emcN3: 0 # Emergency service support indicator for Non-3GPP access (uinteger, range: 0~1)
        mcsi: 0 # MCS indicator (uinteger, range: 0~1)
      t3502Value: 720
      t3512Value: 3600
      non3gppDeregistrationTimerValue: 3240
      # retransmission timer for paging message
      t3513:
        enable: true     # true or false
        expireTime: 6s   # default is 6 seconds
        maxRetryTimes: 4 # the max number of retransmission
      # retransmission timer for NAS Registration Accept message
      t3522:
        enable: true     # true or false
        expireTime: 6s   # default is 6 seconds
        maxRetryTimes: 4 # the max number of retransmission
      # retransmission timer for NAS Registration Accept message
      t3550:
        enable: true     # true or false
        expireTime: 6s   # default is 6 seconds
        maxRetryTimes: 4 # the max number of retransmission
      # retransmission timer for NAS Authentication Request/Security Mode Command message
      t3560:
        enable: true     # true or false
        expireTime: 6s   # default is 6 seconds
        maxRetryTimes: 4 # the max number of retransmission
      # retransmission timer for NAS Notification message
      t3565:
        enable: true     # true or false
        expireTime: 6s   # default is 6 seconds
        maxRetryTimes: 4 # the max number of retransmission
      t3570:
        enable: true     # true or false
        expireTime: 6s   # default is 6 seconds
        maxRetryTimes: 4 # the max number of retransmission

    logger:
      AMF:
        ReportCaller: false
        debugLevel: info
      Aper:
        ReportCaller: false
        debugLevel: info
      FSM:
        ReportCaller: false
        debugLevel: info
      NAS:
        ReportCaller: false
        debugLevel: info
      NGAP:
        ReportCaller: false
        debugLevel: info