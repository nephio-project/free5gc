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

package amf

import (
	"bytes"
	"text/template"
)

const configurationTemplateSource = `
info:

  version: 1.0.3
  description: AMF initial local configuration

configuration:

  ngapIpList:
    - {{ .N2_IP }}

  sbi:
    scheme: http
    registerIPv4: {{ .SVC_NAME }}
    bindingIPv4: 0.0.0.0  # IP used to bind the service
    port: 80
    tls:
      key: config/TLS/amf.key
      pem: config/TLS/amf.pem

  nrfUri: http://nrf-nnrf:8000

  amfName: AMF

  serviceNameList:
  - namf-comm
  - namf-evts
  - namf-mt
  - namf-loc
  - namf-oam

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
 `

var configurationTemplate = template.Must(template.New("AMFConfiguration").Parse(configurationTemplateSource))

type configurationTemplateValues struct {
	SVC_NAME string
	N2_IP    string
}

func renderConfigurationTemplate(values configurationTemplateValues) (string, error) {
	var buffer bytes.Buffer
	if err := configurationTemplate.Execute(&buffer, values); err == nil {
		return buffer.String(), nil
	} else {
		return "", err
	}
}
