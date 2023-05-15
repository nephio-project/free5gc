@@ -0,0 +1,21 @@
/*
 */

package controllers

var AMFCfgTemplate string = `
info:
  version: 1.0.0
  description: AMF configuration
configuration:
  ReportCaller: false
  debugLevel: info
  n2:
    - addr: {{ .N2_IP }}
  n11:
    - addr: {{ .N11_IP }}
