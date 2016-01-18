// tx
package lacp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"net"
)

// bridge will simulate communication between two channels
type SimulationBridge struct {
	port1       uint16
	port2       uint16
	rxLacpPort1 chan gopacket.Packet
	rxLacpPort2 chan gopacket.Packet
	rxLampPort1 chan gopacket.Packet
	rxLampPort2 chan gopacket.Packet
}

func (bridge *SimulationBridge) TxViaGoChannel(port uint16, pdu interface{}) {

	var p *LaAggPort
	if LaFindPortById(port, &p) {

		// Set up all the layers' fields we can.
		eth := layers.Ethernet{
			SrcMAC:       net.HardwareAddr{0x00, uint8(p.PortNum & 0xff), 0x00, 0x01, 0x01, 0x01},
			DstMAC:       layers.SlowProtocolDMAC,
			EthernetType: layers.EthernetTypeSlowProtocol,
		}

		slow := layers.SlowProtocol{
			SubType: layers.SlowProtocolTypeLACP,
		}

		lacp := pdu.(*layers.LACP)

		// Set up buffer and options for serialization.
		buf := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			FixLengths:       true,
			ComputeChecksums: true,
		}

		gopacket.SerializeLayers(buf, opts, &eth, &slow, lacp)
		pkt := gopacket.NewPacket(buf.Bytes(), layers.LinkTypeEthernet, gopacket.Default)

		if port != bridge.port1 && bridge.rxLacpPort1 != nil {
			//fmt.Println("TX channel: Tx From port", port, "bridge Port Rx", bridge.port1)
			//fmt.Println("TX:", pkt)
			bridge.rxLacpPort1 <- pkt
		} else if bridge.rxLacpPort2 != nil {
			//fmt.Println("TX channel: Tx From port", port, "bridge Port Rx", bridge.port2)
			bridge.rxLacpPort2 <- pkt
		}
	} else {
		fmt.Println("Unable to find port in tx")
	}
}

func TxViaLinuxIf(port uint16, pdu interface{}) {
	var p *LaAggPort
	if LaFindPortById(port, &p) {

		txIface, err := net.InterfaceByName(p.IntfNum)

		if err == nil {
			// conver the packet to a go packet
			// Set up all the layers' fields we can.
			eth := layers.Ethernet{
				SrcMAC:       txIface.HardwareAddr,
				DstMAC:       layers.SlowProtocolDMAC,
				EthernetType: layers.EthernetTypeSlowProtocol,
			}

			slow := layers.SlowProtocol{
				SubType: layers.SlowProtocolTypeLACP,
			}

			lacp := pdu.(*layers.LACP)

			// Set up buffer and options for serialization.
			buf := gopacket.NewSerializeBuffer()
			opts := gopacket.SerializeOptions{
				FixLengths:       true,
				ComputeChecksums: true,
			}
			// Send one packet for every address.
			gopacket.SerializeLayers(buf, opts, &eth, &slow, lacp)
			if err := p.handle.WritePacketData(buf.Bytes()); err != nil {
				p.LacpDebug.logger.Info(fmt.Sprintf("%s\n", err))
			}
		} else {
			fmt.Println("ERROR could not find interface", p.IntfNum, err)
		}
	} else {
		fmt.Println("Unable to find port", port)
	}
}
