/*

 */

package controllers

var UPFCfg string = `
info:
  version: 1.0.0
  description: UPF configuration

configuration:
  ReportCaller: false
  debugLevel: info
  dnn_list:
  - cidr: $DNN_CIDR
    dnn: $DNN
    natifname: n6

  pfcp:
    - addr: $PFCP_IP

  gtpu:
    - addr: $GTPU_IP
    # [optional] gtpu.name
    # - name: upf.5gc.nctu.me
    # [optional] gtpu.ifname
    # - ifname: gtpif
`
