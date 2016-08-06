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
package drcp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"l2/lacp/protocol/utils"
	"net"
	"reflect"
)

const RxModuleStr = "Rx Module"

// LaRxMain will process incomming packets from
// a socket as of 10/22/15 packets recevied from
// channel
func DrRxMain(pId uint16, portaladdr string, rxPktChan chan gopacket.Packet) {
	// can be used by test interface
	go func(portId uint16, pa string, rx chan gopacket.Packet) {
		rxMainPort := portId
		rxMainChan := rx
		rxMainDrPortalAddr := pa
		for {
			select {
			case packet, ok := <-rxMainChan:
				//fmt.Println("RxMain: port", rxMainPort)

				if ok {
					//fmt.Println("RxMain: port", rxMainPort)
					//fmt.Println("RX:", packet)

					if isdrcp := IsControlFrame(rxMainPort, packet); isdrcp {
						if isdrcp {
							drcpLayer := packet.Layer(layers.LayerTypeDRCP)
							if drcpLayer == nil {
								fmt.Println("Received non DRCP frame", packet)
							} else {

								// lacp data
								drcp := drcpLayer.(*layers.DRCP)

								ProcessDrcpFrame(rxMainPort, rxMainDrPortalAddr, drcp)
							}
						}
					}
				} else {
					return
				}
			}
		}

		fmt.Println("RX go routine end")
	}(pId, portaladdr, rxPktChan)
}

func IsControlFrame(pId uint16, packet gopacket.Packet) bool {

	drcp := false
	ethernetLayer := packet.Layer(layers.LayerTypeEthernet)
	drcpLayer := packet.Layer(layers.LayerTypeDRCP)
	if ethernetLayer == nil {
		return false
	}

	ethernet := ethernetLayer.(*layers.Ethernet)
	drcpProtocolMACs := make([]net.HardwareAddr, 3)
	drcpProtocolMACs = append(drcpProtocolMACs, net.HardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x00})
	drcpProtocolMACs = append(drcpProtocolMACs, net.HardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x03})
	drcpProtocolMACs = append(drcpProtocolMACs, net.HardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x0E})

	isDrcpProtocolEtherType := ethernet.EthernetType == layers.EthernetTypeDRCP

	// check the mac address and ethertype
	if reflect.DeepEqual(ethernet.DstMAC, drcpProtocolMACs[0]) ||
		reflect.DeepEqual(ethernet.DstMAC, drcpProtocolMACs[1]) ||
		reflect.DeepEqual(ethernet.DstMAC, drcpProtocolMACs[2]) {
		if isDrcpProtocolEtherType &&
			drcpLayer != nil {
			drcp = true
		}
	}

	return drcp
}

// ProcessDrcpFrame will lookup the cooresponding port from which the
// packet arrived and forward the packet to the Rx Machine for processing
func ProcessDrcpFrame(pId uint16, pa string, drcp *layers.DRCP) {
	netAddr := net.HardwareAddr{
		drcp.PortalInfo.PortalAddr[0],
		drcp.PortalInfo.PortalAddr[1],
		drcp.PortalInfo.PortalAddr[2],
		drcp.PortalInfo.PortalAddr[3],
		drcp.PortalInfo.PortalAddr[4],
		drcp.PortalInfo.PortalAddr[5],
	}

	if pa != netAddr.String() {
		// not meant for this portal disregarding
		return
	}

	var dr *DistributedRelay
	if DrFindByPortalAddr(netAddr.String(), &dr) {
		for _, ipp := range dr.Ipplinks {
			if ipp.Id == uint32(pId) {
				ipp.RxMachineFsm.RxmPktRxEvent <- RxDrcpPdu{
					pdu: drcp,
					src: RxModuleStr,
				}
				return
			}
		}
		utils.GlobalLogger.Warning(fmt.Sprintf("RX: Received DRCP Packet on invalid Port %s with Portal Addr %s", pId, pa))
	}
}
