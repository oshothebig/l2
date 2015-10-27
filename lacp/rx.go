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
	metadata  RxPacketMetaData
	dmac      [6]uint8
	smac      [6]uint8
	etherType uint16
	// total size should be 110
	// version 2 lacp not supported "yet"
	lacp     LacpPdu
	reserved [52]uint8
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

				if ok {
					// check if this is a valid frame to process
					if marker, lacp := IsControlFrame(&packet); lacp || marker {

						if lacp {
							ProcessLacpFrame(&packet.metadata, packet.lacp)
						} else if marker {
							// marker protocol
							ProcessLampFrame(&packet.metadata, packet.lacp)
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

	if pdu.dmac[0] == SlowProtocolDmacByte0 &&
		pdu.dmac[1] == SlowProtocolDmacByte1 &&
		pdu.dmac[2] == SlowProtocolDmacByte2 &&
		pdu.dmac[3] == SlowProtocolDmacByte3 &&
		pdu.dmac[4] == SlowProtocolDmacByte4 &&
		pdu.dmac[5] == SlowProtocolDmacByte5 &&
		pdu.etherType == SlowProtocolEtherType &&
		pdu.lacp.subType == LacpSubType {
		lacp = true
		// only supporting marker information
	} else if pdu.dmac[0] == SlowProtocolDmacByte0 &&
		pdu.dmac[1] == SlowProtocolDmacByte1 &&
		pdu.dmac[2] == SlowProtocolDmacByte2 &&
		pdu.dmac[3] == SlowProtocolDmacByte3 &&
		pdu.dmac[4] == SlowProtocolDmacByte4 &&
		pdu.dmac[5] == SlowProtocolDmacByte5 &&
		pdu.etherType == SlowProtocolEtherType &&
		pdu.lacp.subType == LampSubType &&
		pdu.lacp.actor.tlv_type == LampMarkerInformation {
		marker = true
	}

	return marker, lacp
}

// ProcessLacpFrame will lookup the cooresponding port from which the
// packet arrived and forward the packet to the Rx Machine for processing
func ProcessLacpFrame(metadata *RxPacketMetaData, pdu interface{}) {
	var p *LaAggPort
	lacppdu := pdu.(*LacpPdu)

	// lets find the port and only process it if the
	// begin state has been met
	if LaFindPortById(metadata.port, p) && p.begin {
		// lets offload the packet to another thread
		p.RxMachineFsm.RxmPktRxEvent <- LacpRxLacpPdu{
			pdu: lacppdu,
			src: RxModuleStr}
	}
}

func ProcessLampFrame(metadata *RxPacketMetaData, pdu interface{}) {
	var p *LaAggPort

	// copying data over to an array, then cast it back
	lamppdu := pdu.(*LaMarkerPdu)

	if LaFindPortById(metadata.port, p) && p.begin {
		// lets offload the packet to another thread
		//p.RxMachineFsm.RxmPktRxEvent <- *lacppdu
		// TODO send packet to marker responder
		fmt.Println(lamppdu)
	}
}
