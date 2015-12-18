// rx will take care of parsing a received frame from a linux socket
// if checks pass then packet will be either passed rx machine or
// marker responder
package lacp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
	"reflect"
)

const RxModuleStr = "Rx Module"

// LaRxMain will process incomming packets from
// a socket as of 10/22/15 packets recevied from
// channel
func LaRxMain(pId uint16, rxPktChan chan gopacket.Packet) {
	// can be used by test interface
	go func(portId uint16, rx chan gopacket.Packet) {
		rxMainPort := portId
		rxMainChan := rx
		// TODO add logic to either wait on a socket or wait on a channel,
		// maybe both?  Can spawn a seperate go thread to wait on a socket
		// and send the packet to this thread
		for {
			select {
			case packet, ok := <-rxMainChan:
				//fmt.Println("RxMain: port", rxMainPort)

				if ok {
					fmt.Println("RxMain: port", rxMainPort)
					fmt.Println("RX:", packet)

					if marker, lacp := IsControlFrame(packet); lacp || marker {
						//fmt.Println("IsControl Frame ", marker, lacp)
						if lacp {
							lacpLayer := packet.Layer(layers.LayerTypeLACP)
							if lacpLayer == nil {
								fmt.Println("Received non LACP frame", packet)
							} else {

								// lacp data
								lacp := lacpLayer.(*layers.LACP)

								ProcessLacpFrame(rxMainPort, lacp)
							}
						} else {
							fmt.Println("Discard Packet not an lacp frame")
							// discard packet
						}
					} else {
						// discard packet
						fmt.Println("Discarding Packet not lacp or marker")
					}
				} else {
					fmt.Println("Channel closed")
					return
				}
			}
		}

		fmt.Println("RX go routine end")
	}(pId, rxPktChan)
}

func IsControlFrame(packet gopacket.Packet) (bool, bool) {
	lacp := false
	marker := false
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	slowProtocolLayer := packet.Layer(layers.LayerTypeSlowProtocol)
	if ethernetLayer == nil || slowProtocolLayer == nil {
		return false, false
	}

	ethernet := ethernetLayer.(*layers.Ethernet)
	slow := slowProtocolLayer.(*layers.SlowProtocol)
	slowProtocolMAC := net.HardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x02}

	if reflect.DeepEqual(ethernet.DstMAC, slowProtocolMAC) &&
		ethernet.EthernetType == layers.EthernetTypeSlowProtocol &&
		slow.SubType == layers.SlowProtocolTypeLACP {
		lacp = true
		// only supporting marker information
	} else if reflect.DeepEqual(ethernet.DstMAC, slowProtocolMAC) &&
		ethernet.EthernetType == layers.EthernetTypeSlowProtocol &&
		slow.SubType == layers.SlowProtocolTypeLAMP {
		marker = true
	}

	return marker, lacp
}

// ProcessLacpFrame will lookup the cooresponding port from which the
// packet arrived and forward the packet to the Rx Machine for processing
func ProcessLacpFrame(pId uint16, lacp *layers.LACP) {
	var p *LaAggPort

	//fmt.Println(lacp)
	// lets find the port via the info in the packet
	if LaFindPortById(pId, &p) {
		//fmt.Println(lacp)
		p.RxMachineFsm.RxmPktRxEvent <- LacpRxLacpPdu{
			pdu: lacp,
			src: RxModuleStr}
	} else {
		fmt.Println("Unable to find port", lacp.Partner.Info.Port, lacp.Partner.Info.PortPri)
	}
}

/*
func ProcessLampFrame(lamppdu *layers.LAMP) {
	var p *LaAggPort

	// copying data over to an array, then cast it back
	lamppdu := pdu.(*LaMarkerPdu)

	// sanity check
	if metadata.port != lamppdu.requestor_port &&
		lamppdu.requestor_port != 0 {
		fmt.Println("RX: WARNING port", metadata, "LAPort", lamppdu.requestor_port, "do not agree")
	}

	if LaFindPortById(metadata.port, &p) && p.begin {
		// lets offload the packet to another thread
		//p.RxMachineFsm.RxmPktRxEvent <- *lacppdu
		// TODO send packet to marker responder
		//fmt.Println(lamppdu)
	}
}
*/
