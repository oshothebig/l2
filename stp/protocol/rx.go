// rx will take care of parsing a received frame from a linux socket
// if checks pass then packet will be either passed rx machine or
// marker responder
package stp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"reflect"
)

const RxModuleStr = "Rx Module STP"

// BpduRxMain will process incomming packets from
func BpduRxMain(pId int32, bId int32, rxPktChan chan gopacket.Packet) {
	// can be used by test interface
	go func(portId int32, bId int32, rx chan gopacket.Packet) {
		rxMainPort := portId
		rxMainBrg := bId
		rxMainChan := rx
		fmt.Println("RxMain START")
		// TODO add logic to either wait on a socket or wait on a channel,
		// maybe both?  Can spawn a seperate go thread to wait on a socket
		// and send the packet to this thread
		for {
			select {
			case packet, ok := <-rxMainChan:
				//fmt.Println("RxMain: port", rxMainPort)

				if ok {
					if packet != nil {

						p := GetBrgPort(rxMainPort, rxMainBrg, packet)
						if p != nil {
							//fmt.Println("RxMain: port", rxMainPort)
							ptype := ValidateBPDUFrame(p, packet)
							//fmt.Println("RX:", packet, ptype)
							if ptype != BPDURxTypeUnknown {
								ProcessBpduFrame(p, ptype, packet)
							}
						}
					}
				} else {
					StpLogger("INFO", "RXMAIN: Channel closed")
					return
				}
			}
		}

		StpLogger("INFO", "RXMAIN go routine end")
	}(pId, bId, rxPktChan)
}

func IsValidStpPort(pId int32) bool {
	for _, p := range PortListTable {
		if p.IfIndex == pId {
			//fmt.Println("IsValidStpPort: Found valid ifindex", p.IfIndex)
			return true
		}
	}
	return false
}

// find proper bridge for the given port
func GetBrgPort(pId int32, bId int32, packet gopacket.Packet) *StpPort {
	var p *StpPort

	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	llcLayer := packet.Layer(layers.LayerTypeLLC)
	//snapLayer := packet.Layer(layers.LayerTypeSNAP)
	bpduLayer := packet.Layer(layers.LayerTypeBPDU)
	pvstLayer := packet.Layer(layers.LayerTypePVST)
	if ethernetLayer == nil ||
		llcLayer == nil ||
		(bpduLayer == nil && pvstLayer == nil) {
		fmt.Println("NOT a valid packet for this module", pId, bId, packet)

	} else {
		pIntf, _ := PortConfigMap[pId]
		ethernet := ethernetLayer.(*layers.Ethernet)
		if ethernet.SrcMAC[0] == pIntf.HardwareAddr[0] &&
			ethernet.SrcMAC[1] == pIntf.HardwareAddr[1] &&
			ethernet.SrcMAC[2] == pIntf.HardwareAddr[2] &&
			ethernet.SrcMAC[3] == pIntf.HardwareAddr[3] &&
			ethernet.SrcMAC[4] == pIntf.HardwareAddr[4] &&
			ethernet.SrcMAC[5] == pIntf.HardwareAddr[5] {
			// lets drop our own packets
			return p
		}
		//fmt.Println("RX:", packet)

		// only process the bpdu if stp is configured
		if IsValidStpPort(pId) {
			vlan := uint16(DEFAULT_STP_BRIDGE_VLAN)
			if pvstLayer != nil {
				pvst := pvstLayer.(*layers.PVST)
				if pvst.ProtocolVersionId == layers.PVSTProtocolVersion {
					vlan = pvst.OriginatingVlan.OrigVlan
				}
			}
			for _, b := range BridgeListTable {
				if b.BrgIfIndex == bId &&
					b.Vlan == vlan &&
					StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
					return p
				}
			}
		}
	}
	return p
}

// ValidateBPDUFrame: 802.1D Section 9.3.4
// Function shall validate the received BPDU
func ValidateBPDUFrame(p *StpPort, packet gopacket.Packet) (bpduType BPDURxType) {

	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	bpduLayer := packet.Layer(layers.LayerTypeBPDU)
	pvstLayer := packet.Layer(layers.LayerTypePVST)
	ethernet := ethernetLayer.(*layers.Ethernet)

	isBPDUProtocolMAC := reflect.DeepEqual(ethernet.DstMAC, layers.BpduDMAC)
	isPVSTProtocolMAC := reflect.DeepEqual(ethernet.DstMAC, layers.BpduPVSTDMAC)
	fmt.Println("IsBPDU or IsPVST MAC", isBPDUProtocolMAC, isPVSTProtocolMAC)
	if isBPDUProtocolMAC {
		// lets get the actual type of BPDU
		subLayerType := bpduLayer.LayerContents()[3]
		if subLayerType == layers.BPDUTypeSTP {
			stp := bpduLayer.(*layers.STP)
			if len(stp.Contents) >= layers.BPDUTopologyLength &&
				stp.BPDUType == layers.BPDUTypeSTP {
				// condition 9.3.4 (a)
				if stp.ProtocolId == layers.RSTPProtocolIdentifier &&
					len(stp.Contents) >= layers.STPProtocolLength &&
					stp.MsgAge < stp.MaxAge &&
					stp.BridgeId != p.b.BridgePriority.DesignatedBridgeId &&
					stp.PortId != uint16(p.PortId|p.Priority<<8) {

					// Found that Cisco send dot1d frame for tc going to
					// still interpret this as RSTP frame
					if StpGetBpduTopoChange(uint8(stp.Flags)) ||
						StpGetBpduTopoChangeAck(uint8(stp.Flags)) {
						bpduType = BPDURxTypeRSTP
					} else {
						bpduType = BPDURxTypeSTP
					}
				}
			} else {
				bpduType = BPDURxTypeUnknownBPDU
			}
		} else if subLayerType == layers.BPDUTypeRSTP {
			rstp := bpduLayer.(*layers.RSTP)
			// condition 9.3.4 (c)
			if len(rstp.Contents) >= layers.BPDUTopologyLength &&
				rstp.ProtocolId == layers.RSTPProtocolIdentifier {
				// condition 9.3.4 (a)
				if rstp.BPDUType == layers.BPDUTypeRSTP {
					bpduType = BPDURxTypeRSTP
				}
			} else {
				bpduType = BPDURxTypeUnknownBPDU
			}
		} else if subLayerType == layers.BPDUTypeTopoChange {
			topo := bpduLayer.(*layers.BPDUTopology)
			// condition 9.3.4 (b)
			if len(topo.Contents) >= layers.BPDUTopologyLength &&
				topo.ProtocolId == layers.RSTPProtocolIdentifier {
				if topo.BPDUType == layers.BPDUTypeTopoChange {
					bpduType = BPDURxTypeTopo
				}
			} else {
				bpduType = BPDURxTypeUnknownBPDU
			}
		} else {
			bpduType = BPDURxTypeUnknownBPDU
		}
	} else if isPVSTProtocolMAC {
		pvst := pvstLayer.(*layers.PVST)
		if len(pvst.Contents) >= layers.BPDUTopologyLength &&
			pvst.ProtocolId == layers.RSTPProtocolIdentifier {
			// condition 9.3.4 (a)
			if pvst.BPDUType == layers.BPDUTypePVST &&
				len(pvst.Contents) >= layers.PVSTProtocolLength {
				bpduType = BPDURxTypePVST
			} else {
				bpduType = BPDURxTypeUnknownBPDU
			}
		}
	}

	return bpduType
}

// ProcessBpduFrame will lookup the cooresponding port from which the
// packet arrived and forward the packet to the Port Rx Machine for processing
func ProcessBpduFrame(p *StpPort, ptype BPDURxType, packet gopacket.Packet) {

	bpduLayer := packet.Layer(layers.LayerTypeBPDU)

	//fmt.Printf("ProcessBpduFrame on port/bridge\n", pId, bId)
	//fmt.Printf("ProcessBpduFrame %T\n", bpduLayer)
	// lets find the port via the info in the packet
	p.RcvdBPDU = true
	//fmt.Println("Sending rx message to Port Rcvd State Machine", p.IfIndex, p.BrgIfIndex)
	if p.PrxmMachineFsm != nil {
		p.PrxmMachineFsm.PrxmRxBpduPkt <- RxBpduPdu{
			pdu:   bpduLayer, // this is a pointer
			ptype: ptype,
			src:   RxModuleStr}
	} else {
		StpLogger("ERROR", fmt.Sprintf("RXMAIN: rcvd FSM not running %d\n", p.IfIndex))
	}
}
