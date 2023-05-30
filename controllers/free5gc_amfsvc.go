package controllers

var AMFService string = `
info:
  version: 1.0.2
  description: AMF service
 
service:
  serviceNameList:
    - amf-namf
	
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: free5gc-amf
    app.kubernetes.io/instance: amf
    project: free5gc
    nf: amf
`
