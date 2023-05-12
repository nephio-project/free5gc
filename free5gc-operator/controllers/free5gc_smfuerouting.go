package controllers

var Uerouting string = `info:
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