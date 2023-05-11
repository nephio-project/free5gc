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
  pfcp:
    - addr: {{ .PFCP_IP }}


uerouting.yaml: |
    info:
      version: 1.0.1
      description: Routing information for UE
    ueRoutingInfo:
      UE1: # Group Name
        members:
        - imsi-208930000000003 # Subscription Permanent Identifier of the UE
        topology: # Network topology for this group (Uplink: A->B, Downlink: B->A)
        # default path derived from this topology
        # node name should be consistent with smfcfg.yaml
          - A: gNB1
            B: BranchingUPF
          - A: BranchingUPF
            B: AnchorUPF1
        specificPath:
          - dest: 10.100.100.26/32 # the destination IP address on Data Network (DN)
            # the order of UPF nodes in this path. We use the UPF's name to represent each UPF node.
            # The UPF's name should be consistent with smfcfg.yaml
            path: [BranchingUPF, AnchorUPF2]
      
      UE2: # Group Name
        members:
        - imsi-208930000000004 # Subscription Permanent Identifier of the UE
        topology: # Network topology for this group (Uplink: A->B, Downlink: B->A)
        # default path derived from this topology
        # node name should be consistent with smfcfg.yaml
          - A: gNB1
            B: BranchingUPF
          - A: BranchingUPF
            B: AnchorUPF1
        specificPath:
          - dest: 10.100.100.16/32 # the destination IP address on Data Network (DN)
            # the order of UPF nodes in this path. We use the UPF's name to represent each UPF node.
            # The UPF's name should be consistent with smfcfg.yaml
            path: [BranchingUPF, AnchorUPF2]
`
