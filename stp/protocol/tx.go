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

func (p *StpPort) TxRSTP() {

	pIntf, _ := PortConfigMap[p.IfIndex]

	eth, llc := p.BuildRSTPEthernetLlcHeaders()

	rstp := layers.RSTP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: p.BridgeProtocolVersionGet(),
		BPDUType:          byte(layers.BPDUTypeRSTP),
		Flags:             0,
		RootId:            p.DesignatedPriority.RootBridgeId,
		RootPathCost:      uint32(p.DesignatedPriority.RootPathCost),
		BridgeId:          p.DesignatedPriority.DesignatedBridgeId,
		PortId:            uint16(p.DesignatedPriority.DesignatedPortId),
		MsgAge:            uint16(p.DesignatedTimes.MessageAge),
		MaxAge:            uint16(p.DesignatedTimes.MaxAge),
		HelloTime:         uint16(p.DesignatedTimes.HelloTime),
		FwdDelay:          uint16(p.DesignatedTimes.ForwardingDelay),
		Version1Length:    0,
	}

	StpSetBpduFlags(ConvertBoolToUint8(false),
		ConvertBoolToUint8(p.Agree),
		ConvertBoolToUint8(p.Forward),
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

	StpLogger("INFO", fmt.Sprintf("Sent RSTP packet on interface %s\n", pIntf.Name))
}

func (p *StpPort) TxTCN() {
	pIntf, _ := PortConfigMap[p.IfIndex]

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
	StpLogger("INFO", fmt.Sprintf("Sent TCN packet on interface %s\n", pIntf.Name))

}

func (p *StpPort) TxConfig() {
	pIntf, _ := PortConfigMap[p.IfIndex]

	eth, llc := p.BuildRSTPEthernetLlcHeaders()

	stp := layers.STP{
		ProtocolId:        layers.RSTPProtocolIdentifier,
		ProtocolVersionId: layers.STPProtocolVersion,
		BPDUType:          byte(layers.BPDUTypeSTP),
		Flags:             0,
		RootId:            p.DesignatedPriority.RootBridgeId,
		RootPathCost:      uint32(p.DesignatedPriority.RootPathCost),
		BridgeId:          p.DesignatedPriority.DesignatedBridgeId,
		PortId:            uint16(p.DesignatedPriority.DesignatedPortId),
		MsgAge:            uint16(p.DesignatedTimes.MessageAge),
		MaxAge:            uint16(p.DesignatedTimes.MaxAge),
		HelloTime:         uint16(p.DesignatedTimes.HelloTime),
		FwdDelay:          uint16(p.DesignatedTimes.ForwardingDelay),
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
	StpLogger("INFO", fmt.Sprintf("Sent Config packet on interface %s\n", pIntf.Name))
}
