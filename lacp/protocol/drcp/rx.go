//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  |
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  |
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   |
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  |
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__|
//

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
func DrRxMain(pId uint16, rxPktChan chan gopacket.Packet) {
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
					//fmt.Println("RxMain: port", rxMainPort)
					//fmt.Println("RX:", packet)

					if isdrcp := IsControlFrame(rxMainPort, packet); isdrcp {
						//fmt.Println("IsControl Frame ", marker, lacp)
						if isdrcp {
							drcpLayer := packet.Layer(layers.LayerTypeDRCP)
							if drcpLayer == nil {
								fmt.Println("Received non LACP frame", packet)
							} else {

								// lacp data
								lacp := lacpLayer.(*layers.DRCP)

								ProcessDrcpFrame(rxMainPort, drcp)
							}
						} else {
							fmt.Println("Discard Packet not an drcp frame")
							// discard packet
						}
					} else {
						// discard packet
						fmt.Println("Discarding Packet not lacp or marker", packet)
					}
				} else {
					return
				}
			}
		}

		fmt.Println("RX go routine end")
	}(pId, rxPktChan)
}

func IsControlFrame(pId uint16, packet gopacket.Packet) bool {

	drcp := false
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	drcpLayer := packet.Layer(layers.LayerTypeDRCP)
	if ethernetLayer == nil {
		return false, false
	}

	ethernet := ethernetLayer.(*layers.Ethernet)
	drcpProtocolMACs := make([]net.HardwareAddr, 3)
	drcpProtocolMACs = append(drcpProtocolMACs, net.HwardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x00})
	drcpProtocolMACs = append(drcpProtocolMACs, net.HwardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x03})
	drcpProtocolMACs = append(drcpProtocolMACs, net.HwardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x0E})

	isDrcpProtocolEtherType := ethernet.EthernetType == layers.EthernetTypeDRCP

	// check the mac address and ethertype
	if reflect.DeepEqual(ethernet.DstMAC, drcpProtocolMACs[0]) ||
		reflect.DeepEqual(ethernet.DstMAC, drcpProtocolMACs[1]) ||
		reflect.DeepEqual(ethernet.DstMAC, drcpProtocolMACs[2]) {
		slow := slowProtocolLayer.(*layers.SlowProtocol)
		if isDrcpProtocolEtherType &&
			drcpLayer != nil {
			drcp = true
		} /*else {

			// Error cases for stats
			if LaFindPortById(pId, &p) {
				// 802.1ax-2014 7.3.3.1.5
				// TODO Will need a way to know if a packet is picked up by
				// another protocol for valid subtypes
				// NOT handling 50 counters per second rate
				if (!isSlowProtocolMAC &&
					isSlowProtocolEtherType) ||
					(isSlowProtocolMAC &&
						!isSlowProtocolEtherType) {
					p.LacpCounter.AggPortStatsUnknownRx += 1
				}
			}
		}*/
	} /*else {
		if LaFindPortById(pId, &p) {
			// 802.1ax-2014 7.3.3.1.6
			if isSlowProtocolMAC &&
				isSlowProtocolEtherType {
				p.LacpCounter.AggPortStatsIllegalRx += 1
			}
		}
	}*/

	return drcp
}

// ProcessDrcpFrame will lookup the cooresponding port from which the
// packet arrived and forward the packet to the Rx Machine for processing
func ProcessDrcpFrame(pId uint16, drcp *layers.DRCP) {
	/* TODO need lookup for correct DR IPL port
	var p *LaAggPort

	//fmt.Println(lacp)
	// lets find the port via the info in the packet
	if LaFindPortById(pId, &p) {
		//fmt.Println(lacp)
		p.RxMachineFsm.RxmPktRxEvent <- RxDrcpPdu{
			pdu: drcp,
			src: RxModuleStr}
	} else {
		fmt.Println("DRCP: Unable to find port", pId)
	}
	*/
}
