package packet

import (
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"l2/lldp/utils"
	"net"
)

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func TxInit(interval, hold int) *TX {
	var err error
	/*  Set tx interval during init or update
	 *  default value is 30
	 *  Set tx hold multiplier during init or update
	 *  default value is 4
	 */
	txInfo := &TX{
		MessageTxInterval:       interval,
		MessageTxHoldMultiplier: hold,
		useCacheFrame:           false,
	}
	/*  Set TTL Value at the time of init or update of lldp config
	 *  default value comes out to be 120
	 */
	txInfo.ttl = Min(LLDP_MAX_TTL, txInfo.MessageTxInterval*
		txInfo.MessageTxHoldMultiplier)
	txInfo.DstMAC, err = net.ParseMAC(LLDP_PROTO_DST_MAC)
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("parsing lldp protocol Mac failed",
			err))
	}

	return txInfo
}

/*  Function to send out lldp frame to peer on timer expiry.
 *  if a cache entry is present then use that otherwise create a new lldp frame
 *  A new frame will be constructed:
 *		1) if it is first time send
 *		2) if there is config object update
 */
func (gblInfo *TX) SendFrame(macaddr string, port string) []byte {
	temp := make([]byte, 0)
	// if cached then directly send the packet
	if gblInfo.useCacheFrame {
		return gblInfo.cacheFrame
	} else {
		srcmac, _ := net.ParseMAC(macaddr)
		// we need to construct new lldp frame based of the information that we
		// have collected locally
		// Chassis ID: Mac Address of Port
		// Port ID: Port Name
		// TTL: calculated during port init default is 30 * 4 = 120
		payload := gblInfo.createPayload(srcmac, port)
		if payload == nil {
			debug.Logger.Err("Creating payload failed for port " +
				port)
			gblInfo.useCacheFrame = false
			return temp
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
		gblInfo.cacheFrame = make([]byte, len(pkt))
		copied := copy(gblInfo.cacheFrame, pkt)
		if copied < len(pkt) {
			debug.Logger.Err("Cache cannot be created")
			gblInfo.cacheFrame = nil
			gblInfo.useCacheFrame = false
			// return should never happen
			return temp
		}
		debug.Logger.Info("Send Frame is cached")
		gblInfo.useCacheFrame = true
		return pkt
	}
}

/*  helper function to create payload from lldp frame struct
 */
func (gblInfo *TX) createPayload(srcmac []byte, port string) []byte {
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
			debug.Logger.Info(fmt.Sprintln("Chassis id tlv", tlv))

		case 2: // Port ID
			tlv.Type = layers.LLDPTLVPortID
			tlv.Value = EncodeMandatoryTLV(byte(
				layers.LLDPPortIDSubtypeIfaceName),
				[]byte(port))
			debug.Logger.Info(fmt.Sprintln("Port id tlv", tlv))

		case 3: // TTL
			tlv.Type = layers.LLDPTLVTTL
			tb := []byte{0, 0}
			binary.BigEndian.PutUint16(tb, uint16(gblInfo.ttl))
			tlv.Value = append(tlv.Value, tb...)
			debug.Logger.Info(fmt.Sprintln("TTL tlv", tlv))

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
func EncodeTLV(tlv *layers.LinkLayerDiscoveryValue) []byte {

	// copy value into b
	// type : 7 bits
	// leng : 9 bits
	// value: N bytes
	typeLen := uint16(tlv.Type)<<9 | tlv.Length
	temp := make([]byte, 2+len(tlv.Value))
	binary.BigEndian.PutUint16(temp[0:2], typeLen)
	copy(temp[2:], tlv.Value)
	return temp
}

func (t *TX) SetCache(use bool) {
	t.useCacheFrame = use
}

/*  We have deleted the pcap handler and hence we will invalid the cache buffer
 */
func (gblInfo *TX) DeleteCacheFrame() {
	gblInfo.useCacheFrame = false
	gblInfo.cacheFrame = nil
}

/*  Stop Send Tx timer... as we have already delete the pcap handle
 */
func (gblInfo *TX) StopTxTimer() {
	if gblInfo.TxTimer != nil {
		gblInfo.TxTimer.Stop()
	}
}
