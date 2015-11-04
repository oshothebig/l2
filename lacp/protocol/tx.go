// tx
package lacp

import (
//"fmt"
)

// bridge will simulate communication between two channels
type SimulationBridge struct {
	port1       uint16
	port2       uint16
	rxLacpPort1 chan RxPacket
	rxLacpPort2 chan RxPacket
	rxLampPort1 chan RxPacket
	rxLampPort2 chan RxPacket
}

func (bridge *SimulationBridge) TxViaGoChannel(port uint16, pdu interface{}) {

	//fmt.Println("TX channel: Rx port", port, "bridge Port", bridge.port1)
	if port == bridge.port1 {
		lacp := pdu.(*EthernetLacpFrame)
		if lacp.lacp.subType == LacpSubType {
			bridge.rxLacpPort2 <- RxPacket{metadata: RxPacketMetaData{port: bridge.port2},
				pdu: *lacp}
		}
		/*else {
			lamp := pdu.(*EthernetLampFrame)
			bridge.rxLampPort2 <- RxPacket{metadata: RxPacketMetaData{port: bridge.port2},
				lacp: lamp}
		}
		*/
	} else {
		lacp := pdu.(*EthernetLacpFrame)
		if lacp.lacp.subType == LacpSubType {
			bridge.rxLacpPort1 <- RxPacket{metadata: RxPacketMetaData{port: bridge.port1},
				pdu: *lacp}
		}
		/*
			else {
				lamp := pdu.(*EthernetLampFrame)
				bridge.txLampPort1 <- *lamp
			}
		*/
	}
}

func TxViaLinuxIf(port uint16, pdu interface{}) {}
