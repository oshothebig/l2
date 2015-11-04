// rx will take care of parsing a received frame from a linux socket
// if checks pass then packet will be either passed rx machine or
// marker responder
package lacp

import (
	"fmt"
)

const RxModuleStr = "Rx Module"

type RxPacketMetaData struct {
	port uint16
	intf string
}

// These are the only parameters we care about at this time
// Parameters accomidate
type RxPacket struct {
	metadata RxPacketMetaData
	pdu      EthernetLacpFrame
}

// LaRxMain will process incomming packets from
// a socket as of 10/22/15 packets recevied from
// channel
func LaRxMain(rxPktChan chan RxPacket) {

	go func() {
		// TODO add logic to either wait on a socket or wait on a channel,
		// maybe both?  Can spawn a seperate go thread to wait on a socket
		// and send the packet to this thread
		for {
			select {
			case packet, ok := <-rxPktChan:
				//fmt.Println("RX PACKET", packet)
				if ok {
					// check if this is a valid frame to process
					if marker, lacp := IsControlFrame(&packet); lacp || marker {
						//fmt.Println("Control Frame found", lacp, marker)
						if lacp {
							//fmt.Println("ProcessLacpFrame")
							ProcessLacpFrame(&packet.metadata, &packet.pdu.lacp)
						} else if marker {
							// marker protocol
							ProcessLampFrame(&packet.metadata, &packet.pdu.lacp)
						} else {
							// discard packet
						}
					} else {
						// discard packet
					}
				} else {
					return
				}
			}
		}
	}()
}

func IsControlFrame(pdu *RxPacket) (bool, bool) {
	lacp := false
	marker := false

	if pdu.pdu.dmac[0] == SlowProtocolDmacByte0 &&
		pdu.pdu.dmac[1] == SlowProtocolDmacByte1 &&
		pdu.pdu.dmac[2] == SlowProtocolDmacByte2 &&
		pdu.pdu.dmac[3] == SlowProtocolDmacByte3 &&
		pdu.pdu.dmac[4] == SlowProtocolDmacByte4 &&
		pdu.pdu.dmac[5] == SlowProtocolDmacByte5 &&
		pdu.pdu.ethType == SlowProtocolEtherType &&
		pdu.pdu.lacp.subType == LacpSubType {
		lacp = true
		// only supporting marker information
	} else if pdu.pdu.dmac[0] == SlowProtocolDmacByte0 &&
		pdu.pdu.dmac[1] == SlowProtocolDmacByte1 &&
		pdu.pdu.dmac[2] == SlowProtocolDmacByte2 &&
		pdu.pdu.dmac[3] == SlowProtocolDmacByte3 &&
		pdu.pdu.dmac[4] == SlowProtocolDmacByte4 &&
		pdu.pdu.dmac[5] == SlowProtocolDmacByte5 &&
		pdu.pdu.ethType == SlowProtocolEtherType &&
		pdu.pdu.lacp.subType == LampSubType &&
		pdu.pdu.lacp.actor.tlv_type == LampMarkerInformation {
		marker = true
	}

	return marker, lacp
}

// ProcessLacpFrame will lookup the cooresponding port from which the
// packet arrived and forward the packet to the Rx Machine for processing
func ProcessLacpFrame(metadata *RxPacketMetaData, pdu interface{}) {
	var p *LaAggPort
	lacppdu := pdu.(*LacpPdu)

	// sanity check
	if metadata.port != lacppdu.partner.info.port &&
		lacppdu.partner.info.port != 0 {
		fmt.Println("RX: WARNING port", metadata, "LAPort", lacppdu.partner.info.port, "do not agree")
	}

	// lets find the port and only process it if the
	// begin state has been met
	if LaFindPortById(metadata.port, &p) {
		//fmt.Println("Sending Pkt to Rx Machine")
		// lets offload the packet to another thread
		p.RxMachineFsm.RxmPktRxEvent <- LacpRxLacpPdu{
			pdu: lacppdu,
			src: RxModuleStr}
	} else {
		fmt.Println("Unable to find port", metadata.port)
	}

}

func ProcessLampFrame(metadata *RxPacketMetaData, pdu interface{}) {
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
