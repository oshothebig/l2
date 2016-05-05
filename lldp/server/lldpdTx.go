package lldpServer

import (
	"encoding/binary"
	"errors"
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

		gblInfo.txTimer = time.NewTimer(time.Duration(gblInfo.lldpMessageTxInterval) *
			time.Second)
		svr.lldpGblInfo[ifIndex] = gblInfo
		// Start lldpMessage Tx interval
		<-gblInfo.txTimer.C

		//	<-time.After(time.Duration(gblInfo.lldpMessageTxInterval) *
		//		time.Second)
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
	} else {
		srcmac, _ := net.ParseMAC(gblInfo.MacAddr)
		// we need to construct new lldp frame based of the information that we
		// have collected locally
		// Chassis ID: Mac Address of Port
		// Port ID: Port Name
		// TTL: calculated during port init default is 30 * 4 = 120
		payload := gblInfo.CreatePayload(srcmac)
		if payload == nil {
			gblInfo.logger.Err("Creating payload failed for port " +
				gblInfo.Name)
			gblInfo.useCacheFrame = false
			return
		}
		// Additional TLV's... @TODO: get it done later on
		// System information... like "show version" command at Cisco
		// System Capabilites...

		// Construct ethernet information
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
		gopacket.SerializeLayers(buffer, options, eth,
			gopacket.Payload(payload))
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
}

/*  helper function to create payload from lldp frame struct
 */
func (gblInfo *LLDPGlobalInfo) CreatePayload(srcmac []byte) []byte {
	//payload := make([]byte, 0, LLDP_MIN_FRAME_LENGTH)
	var payload []byte
	tlvType := 1
	for {
		if tlvType > 3 { // right now only minimal lldp tlv
			break
		}
		tlv := &layers.LinkLayerDiscoveryValue{}
		switch tlvType {
		case 1: // Chassis ID
			tlv.Type = layers.LLDPTLVChassisID
			tlv.Value = EncodeMandatoryTLV(byte(
				layers.LLDPChassisIDSubTypeMACAddr), srcmac)
			gblInfo.logger.Debug(fmt.Sprintln("Chassis id tlv", tlv))

		case 2: // Port ID
			tlv.Type = layers.LLDPTLVPortID
			tlv.Value = EncodeMandatoryTLV(byte(
				layers.LLDPPortIDSubtypeIfaceName),
				[]byte(gblInfo.Name))
			gblInfo.logger.Debug(fmt.Sprintln("Port id tlv", tlv))

		case 3: // TTL
			tlv.Type = layers.LLDPTLVTTL
			tb := []byte{0, 0}
			binary.BigEndian.PutUint16(tb, uint16(gblInfo.ttl))
			tlv.Value = append(tlv.Value, tb...)
			gblInfo.logger.Debug(fmt.Sprintln("TTL tlv", tlv))

			// @TODO: add other cases for optional tlv
		}
		tlv.Length = uint16(len(tlv.Value))
		payload = append(payload, EncodeTLV(tlv)...)
		tlvType++
	}

	// After all TLV's are added we need to go ahead and Add LLDPTLVEnd
	tlv := &layers.LinkLayerDiscoveryValue{}
	tlv.Type = layers.LLDPTLVEnd
	tlv.Length = 0
	payload = append(payload, EncodeTLV(tlv)...)
	return payload
}

/*  Write packet is helper function to send packet on wire.
 *  It will inform caller that packet was send successfully and you can go ahead
 *  and cache the pkt or else do not cache the packet as it is corrupted or there
 *  was some error
 */
func (gblInfo *LLDPGlobalInfo) WritePacket(pkt []byte) bool {
	var err error
	gblInfo.PcapHdlLock.Lock()
	if gblInfo.PcapHandle != nil {
		err = gblInfo.PcapHandle.WritePacketData(pkt)
	} else {
		err = errors.New("Pcap Handle is invalid for " + gblInfo.Name)
	}
	gblInfo.PcapHdlLock.Unlock()
	if err != nil {
		gblInfo.logger.Err(fmt.Sprintln("Sending packet failed Error:",
			err, "for Port:", gblInfo.Name))
		return false
	}
	return true
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
func EncodeTLV(tlv *layers.LinkLayerDiscoveryValue) []byte { //, b []byte) error {

	// copy value into b
	// type : 7 bits
	// leng : 9 bits
	// value: N bytes
	typeLen := uint16(tlv.Type)<<9 | tlv.Length
	//temp := []byte{0, 0}
	temp := make([]byte, 2+len(tlv.Value))
	//b = append(b, temp...)
	//binary.BigEndian.PutUint16(b[len(b)-2:], typeLen)
	binary.BigEndian.PutUint16(temp[0:2], typeLen)
	copy(temp[2:], tlv.Value)
	//b = append(b, tlv.Value...)
	return temp
}
