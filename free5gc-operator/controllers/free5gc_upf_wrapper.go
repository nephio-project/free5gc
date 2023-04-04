/*

 */

package controllers

var UPFWrapperScript string = `#!/bin/sh

### Implement networking rules
iptables -A FORWARD -j ACCEPT
{{- range $dnn := .N6cfg }}
iptables -t nat -A POSTROUTING -s {{ $dnn.UEIPPool }} -o {{ $dnn.Interface.Name }} -j MASQUERADE  # route traffic comming from the UE  SUBNET to the interface N6
{{- end }}
echo "1200 n6if" >> /etc/iproute2/rt_tables # create a routing table for the interface N6
{{- range $idx, $dnn := .N6cfg }}
ip rule add from {{ $dnn.UEIPPool }} table n6if   # use the created ip table to route the traffic comming from  the UE SUBNET
ip route add default via {{index $dnn.Interface.GatewayIPs 0}} dev {{ $dnn.Interface.Name }} table n6if  # add a default route in the created table so  that all UEs will use this gateway for external communications (target IP not in the Data Network attached  to the interface N6) and then the Data Network will manage to route the traffic
{{- end }}

/free5gc/free5gc-upfd/free5gc-upfd -c /free5gc/config//upfcfg.yaml
`
