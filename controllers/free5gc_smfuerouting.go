package controllers

var Uerouting string = `
info:
  version: 1.0.1
  description: Routing information for UE
ueRoutingInfo:
  UE1:
    members:
    - imsi-208930000000003
    topology:
      - A: gNB1
        B: BranchingUPF
      - A: BranchingUPF
        B: AnchorUPF1
    specificPath:
      - dest: 10.100.100.26/32
        path: [BranchingUPF, AnchorUPF2]
  UE2:
    members:
    - imsi-208930000000004
    topology:
      - A: gNB1
        B: BranchingUPF
      - A: BranchingUPF
        B: AnchorUPF1
    specificPath:
      - dest: 10.100.100.16/32
        path: [BranchingUPF, AnchorUPF2]
`
