// tx.go
package stp

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func ConvertBoolToUint8(v bool) (rv uint8) {
	if v {
		rv = 1
	}
	return rv
}

func (p *StpPort) BuildRSTPEthernetLlcHeaders() (eth layers.Ethernet, llc layers.LLC) {
	pIntf, _ := PortConfigMap[p.IfIndex]

	eth = layers.Ethernet{
		SrcMAC: pIntf.HardwareAddr,
		DstMAC: layers.BpduDMAC,
		// length
		EthernetType: layers.EthernetTypeLLC,
		Length:       uint16(layers.STPProtocolLength + 3), // found from PCAP from packetlife.net
	}

	llc = layers.LLC{
		DSAP:    0x42,
		IG:      false,
		SSAP:    0x42,
		CR:      false,
		Control: 0x03,
	}
	return eth, llc
}

func (p *StpPort) TxPVST() {
	pIntf, _ := PortConfigMap[p.IfIndex]

	eth := layers.Ethernet{
		SrcMAC: pIntf.HardwareAddr,
		DstMAC: layers.BpduPVSTDMAC,
		// length
		EthernetType: layers.EthernetTypeDot1Q,
	}

	vlan := layers.Dot1Q{
		Priority:       PVST_VLAN_PRIORITY,
		DropEligible:   false,
		VLANIdentifier: p.b.Vlan,
		Type:           layers.EthernetTypeLLC,
	}

	llc := layers.LLC{
		DSAP:    0xAA,
		IG:      false,
		SSAP:    0xAA,
		CR:      false,
		Control: 0x03,
	}

	snap := layers.SNAP{
		OrganizationalCode: []byte{0x00, 0x00, 0x0C, 0x01, 0x0b},
	}

	pvst := layers.PVST{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: p.BridgeProtocolVersionGet(),
		BPDUType:          byte(layers.BPDUTypeRSTP),
		Flags:             0,
		RootId:            p.PortPriority.RootBridgeId,
		RootPathCost:      uint32(p.b.BridgePriority.RootPathCost),
		BridgeId:          p.b.BridgePriority.DesignatedBridgeId,
		PortId:            uint16(p.PortId | p.Priority<<8),
		MsgAge:            uint16(p.b.RootTimes.MessageAge << 8),
		MaxAge:            uint16(p.b.RootTimes.MaxAge << 8),
		HelloTime:         uint16(p.b.RootTimes.HelloTime << 8),
		FwdDelay:          uint16(p.b.RootTimes.ForwardingDelay << 8),
		//MsgAge:         uint16(p.PortTimes.MessageAge),
		//MaxAge:         uint16(p.PortTimes.MaxAge),
		//HelloTime:      uint16(p.PortTimes.HelloTime),
		//FwdDelay:       uint16(p.PortTimes.ForwardingDelay),
		Version1Length: 0,
		OriginatingVlan: layers.STPOriginatingVlanTlv{
			Type:     0,
			Length:   2,
			OrigVlan: p.b.Vlan,
		},
	}

	StpSetBpduFlags(ConvertBoolToUint8(false),
		ConvertBoolToUint8(p.Agree),
		ConvertBoolToUint8(p.Forwarding),
		ConvertBoolToUint8(p.Learning),
		p.Role,
		ConvertBoolToUint8(p.Proposed),
		ConvertBoolToUint8(p.TcWhileTimer.count != 0),
		&pvst.Flags)

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &vlan, &llc, &snap, &pvst)
	if err := p.handle.WritePacketData(buf.Bytes()); err != nil {
		StpLogger("ERROR", fmt.Sprintf("Error writing packet to interface %s\n", err))
		return
	}
	p.SetTxPortCounters(BPDURxTypePVST)

	//pIntf, _ := PortConfigMap[p.IfIndex]
	//StpLogger("INFO", fmt.Sprintf("Sent RSTP packet on interface %s %#v\n", pIntf.Name, rstp))
}

func (p *StpPort) TxRSTP() {

	if p.b.Vlan != DEFAULT_STP_BRIDGE_VLAN {
		p.TxPVST()
		return
	}

	eth, llc := p.BuildRSTPEthernetLlcHeaders()

	rstp := layers.RSTP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: p.BridgeProtocolVersionGet(),
		BPDUType:          byte(layers.BPDUTypeRSTP),
		Flags:             0,
		RootId:            p.PortPriority.RootBridgeId,
		RootPathCost:      uint32(p.b.BridgePriority.RootPathCost),
		BridgeId:          p.b.BridgePriority.DesignatedBridgeId,
		PortId:            uint16(p.PortId | p.Priority<<8),
		MsgAge:            uint16(p.b.RootTimes.MessageAge << 8),
		MaxAge:            uint16(p.b.RootTimes.MaxAge << 8),
		HelloTime:         uint16(p.b.RootTimes.HelloTime << 8),
		FwdDelay:          uint16(p.b.RootTimes.ForwardingDelay << 8),
		//MsgAge:         uint16(p.PortTimes.MessageAge),
		//MaxAge:         uint16(p.PortTimes.MaxAge),
		//HelloTime:      uint16(p.PortTimes.HelloTime),
		//FwdDelay:       uint16(p.PortTimes.ForwardingDelay),
		Version1Length: 0,
	}

	StpSetBpduFlags(ConvertBoolToUint8(false),
		ConvertBoolToUint8(p.Agree),
		ConvertBoolToUint8(p.Forwarding),
		ConvertBoolToUint8(p.Learning),
		p.Role,
		ConvertBoolToUint8(p.Proposed),
		ConvertBoolToUint8(p.TcWhileTimer.count != 0),
		&rstp.Flags)

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &llc, &rstp)
	if err := p.handle.WritePacketData(buf.Bytes()); err != nil {
		StpLogger("ERROR", fmt.Sprintf("Error writing packet to interface %s\n", err))
		return
	}
	p.SetTxPortCounters(BPDURxTypeRSTP)

	//pIntf, _ := PortConfigMap[p.IfIndex]
	//StpLogger("INFO", fmt.Sprintf("Sent RSTP packet on interface %s %#v\n", pIntf.Name, rstp))
}

func (p *StpPort) TxTCN() {
	eth, llc := p.BuildRSTPEthernetLlcHeaders()

	topo := layers.BPDUTopology{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.STPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeTopoChange),
	}

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &llc, &topo)
	if err := p.handle.WritePacketData(buf.Bytes()); err != nil {
		StpLogger("ERROR", fmt.Sprintf("Error writing packet to interface %s\n", err))
		return
	}

	p.SetTxPortCounters(BPDURxTypeTopo)
	//pIntf, _ := PortConfigMap[p.IfIndex]
	//StpLogger("INFO", fmt.Sprintf("Sent TCN packet on interface %s\n", pIntf.Name))

}

func (p *StpPort) TxConfig() {
	eth, llc := p.BuildRSTPEthernetLlcHeaders()

	stp := layers.STP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.STPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeSTP),
		Flags:             0,
		RootId:            p.PortPriority.RootBridgeId,
		RootPathCost:      uint32(p.b.BridgePriority.RootPathCost),
		BridgeId:          p.b.BridgePriority.DesignatedBridgeId,
		PortId:            uint16(p.PortId | p.Priority<<8),
		MsgAge:            uint16(p.b.RootTimes.MessageAge << 8),
		MaxAge:            uint16(p.b.RootTimes.MaxAge << 8),
		HelloTime:         uint16(p.b.RootTimes.HelloTime << 8),
		FwdDelay:          uint16(p.b.RootTimes.ForwardingDelay << 8),
		//MsgAge:    uint16(p.PortTimes.MessageAge),
		//MaxAge:    uint16(p.PortTimes.MaxAge),
		//HelloTime: uint16(p.PortTimes.HelloTime),
		//FwdDelay:  uint16(p.PortTimes.ForwardingDelay),
	}

	StpSetBpduFlags(ConvertBoolToUint8(p.TcAck),
		ConvertBoolToUint8(p.Agree),
		ConvertBoolToUint8(p.Forward),
		ConvertBoolToUint8(p.Learning),
		p.Role,
		ConvertBoolToUint8(p.Proposed),
		ConvertBoolToUint8(p.TcWhileTimer.count != 0),
		&stp.Flags)

	// Set up buffer and options for serialization.
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	// Send one packet for every address.
	gopacket.SerializeLayers(buf, opts, &eth, &llc, &stp)
	if err := p.handle.WritePacketData(buf.Bytes()); err != nil {
		StpLogger("ERROR", fmt.Sprintf("Error writing packet to interface %s\n", err))
		return
	}

	p.SetTxPortCounters(BPDURxTypeSTP)
	//pIntf, _ := PortConfigMap[p.IfIndex]
	//StpLogger("INFO", fmt.Sprintf("Sent Config packet on interface %s %#v\n", pIntf.Name, stp))
}
