package lldpServer

import (
	"encoding/binary"
	_ "errors"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"net"
	"time"
)

/*  lldp server go routine to handle tx timer... once the timer fires we will
 *  send the ifindex on the channel to handle send info
 */
func (svr *LLDPServer) TransmitFrames(pHandle *pcap.Handle, ifIndex int32) {
	for {
		gblInfo, exists := svr.lldpGblInfo[ifIndex]
		if !exists {
			return
		}
		// Start lldpMessage Tx interval
		<-time.After(time.Duration(gblInfo.lldpMessageTxInterval) *
			time.Second)
		svr.lldpTxPktCh <- SendPktChannel{
			ifIndex: ifIndex,
		}
	}
}

/*  Function to send out lldp frame to peer on timer expiry.
 *  if a cache entry is present then use that otherwise create a new lldp frame
 *  A new frame will be constructed:
 *		1) if it is first time send
 *		2) if there is config object update
 */
func (gblInfo *LLDPGlobalInfo) SendFrame() {
	// if cached then directly send the packet
	if gblInfo.useCacheFrame {
		if cache := gblInfo.WritePacket(gblInfo.cacheFrame); cache == false {
			gblInfo.useCacheFrame = false
		}
	}

	// we need to construct new lldp frame based of the information that we
	// have collected locally
	// Chassis ID: Mac Address of Port
	// Port ID: Port Name
	// TTL: calculated during port init default is 30 * 4 = 120
	txFrame := &layers.LinkLayerDiscovery{
		ChassisID: layers.LLDPChassisID{
			Subtype: layers.LLDPChassisIDSubTypeMACAddr,
			ID:      []byte(gblInfo.MacAddr),
		},
		PortID: layers.LLDPPortID{
			Subtype: layers.LLDPPortIDSubtypeIfaceName,
			ID:      []byte(gblInfo.Name),
		},

		TTL: uint16(gblInfo.ttl),
	}
	/*
		var txFrame []layers.LinkLayerDiscoveryValue
		val := layers.LinkLayerDiscoveryValue{
			Type:  layers.LLDPTLVChassisID,
			Value: []byte(gblInfo.MacAddr),
		}
		val.Length = uint16(1 + len(val.Value))
		txFrame = append(txFrame, val)
	*/
	payload := gblInfo.CreatePayload(txFrame)
	if payload == nil {
		gblInfo.logger.Err("Creating payload failed for port " +
			gblInfo.Name)
		gblInfo.useCacheFrame = false
		return
	}
	// Additional TLV's... @TODO: get it done later on
	// System information... like "show version" command at Cisco
	// System Capabilites...
	//txLinkInfo = &layers.LayerTypeLinkLayerDiscoveryInfo{}

	// Construct ethernet information
	srcmac, _ := net.ParseMAC(gblInfo.MacAddr)
	eth := &layers.Ethernet{
		SrcMAC:       srcmac,
		DstMAC:       gblInfo.DstMAC,
		EthernetType: layers.EthernetTypeLinkLayerDiscovery,
	}

	// construct new buffer
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	gopacket.SerializeLayers(buffer, options, eth, gopacket.Payload(payload))
	pkt := buffer.Bytes()
	cache := gblInfo.WritePacket(pkt)
	if cache {
		gblInfo.cacheFrame = make([]byte, len(pkt))
		copied := copy(gblInfo.cacheFrame, pkt)
		if copied < len(pkt) {
			gblInfo.logger.Err("Cache cannot be created")
			gblInfo.cacheFrame = nil
			gblInfo.useCacheFrame = false
			return
		}
		gblInfo.useCacheFrame = true
	} else {
		gblInfo.useCacheFrame = false
	}
}

/*  helper function to create payload from lldp frame struct
 */
func (gblInfo *LLDPGlobalInfo) CreatePayload(txFrame *layers.LinkLayerDiscovery) []byte {
	payload := make([]byte, gblInfo.Length(txFrame))
	var offset = 0
	cTLV := layers.LinkLayerDiscoveryValue{
		Type: layers.LLDPTLVChassisID,
		Value: EncodeMandatoryTLV(byte(txFrame.ChassisID.Subtype),
			txFrame.ChassisID.ID),
	}
	cTLV.Length = uint16(len(cTLV.Value))
	EncodeTLV(cTLV, payload, &offset)

	pTLV := layers.LinkLayerDiscoveryValue{
		Type: layers.LLDPTLVPortID,
		Value: EncodeMandatoryTLV(byte(txFrame.PortID.Subtype),
			txFrame.PortID.ID),
	}
	pTLV.Length = uint16(len(pTLV.Value))
	EncodeTLV(pTLV, payload, &offset)

	tb := make([]byte, 2)
	binary.BigEndian.PutUint16(tb, txFrame.TTL)
	tTLV := layers.LinkLayerDiscoveryValue{
		Type:   layers.LLDPTLVTTL,
		Length: 2,
		Value:  tb,
	}
	EncodeTLV(tTLV, payload, &offset)

	for _, optTLV := range txFrame.Values {
		err := EncodeTLV(optTLV, payload, &offset)
		if err != nil {
			payload = nil
			break
		}
	}

	return payload
}

/*  Write packet is helper function to send packet on wire.
 *  It will inform caller that packet was send successfully and you can go ahead
 *  and cache the pkt or else do not cache the packet as it is corrupted or there
 *  was some error
 */
func (gblInfo *LLDPGlobalInfo) WritePacket(pkt []byte) bool {
	gblInfo.PcapHdlLock.Lock()
	err := gblInfo.PcapHandle.WritePacketData(pkt)
	gblInfo.PcapHdlLock.Unlock()
	if err != nil {
		gblInfo.logger.Err(fmt.Sprintln("Sending packet failed Error:",
			err, "for Port:", gblInfo.Name))
		return false
	}
	return true
}

/*  return the total bytes of the lldp frame
 */
func (gblInfo *LLDPGlobalInfo) Length(txFrame *layers.LinkLayerDiscovery) int {
	// mandatory tlv's length is always calculated
	var l int
	// Chassis ID Length:
	// 2 byte for type and length
	// 1 byte for chassis id type
	// len(id) for remaining byte
	l += 2 + 1 + len(txFrame.ChassisID.ID)

	// Port ID Length:
	// 2 byte for type and length
	// 1 byte for port id
	// len(id) for remaining byte
	l += 2 + 1 + len(txFrame.PortID.ID)

	// TLV length:
	// 2 byte for type and length
	// 2 byte for uint16
	l += 2 + 2

	// Optional TLV length
	for _, tlv := range txFrame.Values {
		l += 2 + len(tlv.Value)
	}

	// End of LLDP
	l += 2

	return l
}

/*  Encode Mandatory tlv, chassis id and port id
 */
func EncodeMandatoryTLV(Subtype byte, ID []byte) []byte {
	// 1 byte: subtype
	// N bytes: ID
	b := make([]byte, 1+len(ID))
	b[0] = byte(Subtype)
	copy(b[1:], ID)

	return b
}

// Marshall tlv information into binary form
// 1) Check type value
// 2) Check Length
func EncodeTLV(tlv layers.LinkLayerDiscoveryValue, b []byte, offset *int) error {
	/*
		// check type
		if c.Type > TLVTypeMax {
			return LLDP_ERR_INVALID_TLV_TYPE
		}

		// check length
		if int(c.Length) != len(c.Value) {
			return LLDP_ERR_INVALID_TLV_LENGTH
		}
	*/
	// copy value into b
	// type : 7 bits
	// leng : 9 bits
	// value: N bytes

	var typeByte uint16
	typeByte |= uint16(tlv.Type) << 9
	typeByte |= tlv.Length
	binary.BigEndian.PutUint16(b[(0+*offset):(2+*offset)], typeByte)
	copy(b[(2+*offset):(int(tlv.Length)+*offset)], tlv.Value)
	*offset += len(b)
	return nil
}
