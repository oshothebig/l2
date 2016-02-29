// rx will take care of parsing a received frame from a linux socket
// if checks pass then packet will be either passed rx machine or
// marker responder
package stp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
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
						//fmt.Println("RxMain: port", rxMainPort)
						ptype := ValidateBPDUFrame(rxMainPort, rxMainBrg, packet)
						//fmt.Println("RX:", packet, ptype)
						if ptype != BPDURxTypeUnknown {

							ProcessBpduFrame(rxMainPort, rxMainBrg, ptype, packet)
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

// ValidateBPDUFrame: 802.1D Section 9.3.4
// Function shall validate the received BPDU
func ValidateBPDUFrame(pId int32, bId int32, packet gopacket.Packet) (bpduType BPDURxType) {
	var p *StpPort
	bpduType = BPDURxTypeUnknown

	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	llcLayer := packet.Layer(layers.LayerTypeLLC)
	snapLayer := packet.Layer(layers.LayerTypeSNAP)
	bpduLayer := packet.Layer(layers.LayerTypeBPDU)
	pvstLayer := packet.Layer(layers.LayerTypePVST)
	if ethernetLayer == nil ||
		llcLayer == nil ||
		(bpduLayer == nil && pvstLayer == nil) {
		return bpduType
	}

	// only process the bpdu if stp is configured
	if IsValidStpPort(pId) {
		vlan := uint16(DEFAULT_STP_BRIDGE_VLAN)
		if pvstLayer != nil {
			pvst := pvstLayer.(*layers.PVST)
			vlan = pvst.OriginatingVlan.OrigVlan
		}
		for _, b := range BridgeListTable {
			//fmt.Println("ValidateBPDUFrame: Looking for bridge vlan found", bId, vlan, b.BrgIfIndex, b.Vlan)
			if b.BrgIfIndex == bId &&
				b.Vlan == vlan &&
				StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
				//fmt.Println("ValidateBPDUFrame: found stp port", p.IfIndex)

				ethernet := ethernetLayer.(*layers.Ethernet)

				bpduMAC := net.HardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x00}
				pvstMAC := net.HardwareAddr{0x01, 00, 0xCC, 0xCC, 0xCC, 0xCD}
				isBPDUProtocolMAC := reflect.DeepEqual(ethernet.DstMAC, bpduMAC)
				isPVSTProtocolMAC := reflect.DeepEqual(ethernet.DstMAC, pvstMAC)
				//fmt.Println("IsBPDU or IsPVST MAC", isBPDUProtocolMAC, isPVSTProtocolMAC)
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
								stp.BridgeId != p.PortPriority.DesignatedBridgeId &&
								stp.PortId != uint16(p.PortPriority.DesignatedPortId) {
								bpduType = BPDURxTypeSTP
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
				} else if isPVSTProtocolMAC &&
					snapLayer != nil {
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
				break
			}
		}
	} else {
		StpLogger("INFO", fmt.Sprintf("RXMAIN: Unabled to find port %d\n", pId))
	}

	return bpduType
}

// ProcessBpduFrame will lookup the cooresponding port from which the
// packet arrived and forward the packet to the Port Rx Machine for processing
func ProcessBpduFrame(pId int32, bId int32, ptype BPDURxType, packet gopacket.Packet) {
	var p *StpPort

	bpduLayer := packet.Layer(layers.LayerTypeBPDU)

	//fmt.Printf("ProcessBpduFrame %T", bpduLayer)
	//fmt.Printf("ProcessBpduFrame on port", pId)
	// lets find the port via the info in the packet
	vlan := uint16(DEFAULT_STP_BRIDGE_VLAN)
	if ptype == BPDURxTypePVST {
		pvstLayer := packet.Layer(layers.LayerTypePVST)
		pvst := pvstLayer.(*layers.PVST)
		vlan = pvst.OriginatingVlan.OrigVlan
	}
	for _, b := range BridgeListTable {
		if b.BrgIfIndex == bId &&
			b.Vlan == vlan &&
			StpFindPortByIfIndex(pId, b.BrgIfIndex, &p) {
			p.RcvdBPDU = true
			//fmt.Println("Sending rx message to Port Rcvd State Machine", p.IfIndex)
			if p.PrxmMachineFsm != nil {
				p.PrxmMachineFsm.PrxmRxBpduPkt <- RxBpduPdu{
					pdu:   bpduLayer, // this is a pointer
					ptype: ptype,
					src:   RxModuleStr}
			} else {
				StpLogger("ERROR", fmt.Sprintf("RXMAIN: rcvd FSM not running %d\n", pId))
			}
			break
		} else {
			StpLogger("ERROR", fmt.Sprintf("RXMAIN: Unabled to find port %d vlan %d\n", pId, vlan))
		}
	}
}
