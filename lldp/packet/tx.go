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

package packet

import (
	"encoding/binary"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"l2/lldp/utils"
	"models"
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
func (gblInfo *TX) SendFrame(macaddr string, port string, portnum int32, sysInfo *models.SystemParams) []byte { //switchMac, mgmtIp, desc string) []byte {
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
		payload := gblInfo.createPayload(srcmac, port, portnum, sysInfo) // switchMac, mgmtIp, desc)
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
func (gblInfo *TX) createPayload(srcmac []byte, port string, portnum int32, sysInfo *models.SystemParams) []byte { // switchMac, mgmtIp, desc string) []byte {
	var payload []byte
	tlvType := layers.LLDPTLVChassisID // start with chassis id always
	for {
		if tlvType > LLDP_TOTAL_TLV_SUPPORTED { // right now only minimal lldp tlv
			break
		} else if tlvType > layers.LLDPTLVTTL && sysInfo == nil {
			debug.Logger.Info("Reading System Information from DB failed and hence sending out only " +
				"Mandatory TLV's")
			break
		}
		tlv := &layers.LinkLayerDiscoveryValue{}
		switch tlvType {
		case layers.LLDPTLVChassisID: // Chassis ID
			tlv.Type = layers.LLDPTLVChassisID
			tlv.Value = EncodeMandatoryTLV(byte(layers.LLDPChassisIDSubTypeMACAddr), srcmac)
			debug.Logger.Info(fmt.Sprintln("Chassis id tlv", tlv))

		case layers.LLDPTLVPortID: // Port ID
			tlv.Type = layers.LLDPTLVPortID
			tlv.Value = EncodeMandatoryTLV(byte(layers.LLDPPortIDSubtypeIfaceName), []byte(port))
			debug.Logger.Info(fmt.Sprintln("Port id tlv", tlv))

		case layers.LLDPTLVTTL: // TTL
			tlv.Type = layers.LLDPTLVTTL
			tb := []byte{0, 0}
			binary.BigEndian.PutUint16(tb, uint16(gblInfo.ttl))
			tlv.Value = append(tlv.Value, tb...)
			debug.Logger.Info(fmt.Sprintln("TTL tlv", tlv))

		case layers.LLDPTLVPortDescription:
			tlv.Type = layers.LLDPTLVPortDescription
			tlv.Value = []byte("Dummy Port")
			debug.Logger.Info(fmt.Sprintln("Port Description", tlv))

		case layers.LLDPTLVSysDescription:
			tlv.Type = layers.LLDPTLVSysDescription
			tlv.Value = []byte(sysInfo.Description)
			debug.Logger.Info(fmt.Sprintln("System Description", tlv))

		case layers.LLDPTLVSysName:
			tlv.Type = layers.LLDPTLVSysName
			tlv.Value = []byte(sysInfo.Hostname)
			debug.Logger.Info(fmt.Sprintln("System Name", tlv))

		case layers.LLDPTLVMgmtAddress:
			/*
			 *  Value: N bytes
			 *     Subtype is 1 byte
			 *     Address is []byte
			 *     IntefaceSubtype is 1 byte
			 *     IntefaceNumber uint32 <<< this is system interface number which is IfIndex in out case
			 *     OID string
			 */
			mgmtInfo := &layers.LLDPMgmtAddress{
				Subtype:          layers.IANAAddressFamily802,
				Address:          []byte(sysInfo.MgmtIp),
				InterfaceSubtype: layers.LLDPInterfaceSubtypeifIndex,
				InterfaceNumber:  uint32(portnum),
			}
			debug.Logger.Info(fmt.Sprintln("Mgmt Info", mgmtInfo))
			tlv.Type = layers.LLDPTLVMgmtAddress
			tlv.Value = EncodeMgmtTLV(mgmtInfo)
			debug.Logger.Info(fmt.Sprintln("Mgmt Description", tlv))
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

/*  TLV Type = 8, 7 bits             ----
					|--> 2 bytes
 *  TLV Length = 9 bits....	     ----
 *  Value: N bytes
 *     Subtype is 1 byte
 *     Address is []byte
 *     IntefaceSubtype is 1 byte
 *     IntefaceNumber uint32 <<< this is system interface number which is IfIndex in out case
 *     OID string
*/
func EncodeMgmtTLV(tlv *layers.LLDPMgmtAddress) []byte {
	var b []byte
	b = append(b, byte(len(tlv.Address)))
	b = append(b, byte(tlv.Subtype))
	b = append(b, tlv.Address...)
	b = append(b, byte(tlv.InterfaceSubtype))
	b = append(b, byte(tlv.InterfaceNumber))
	b = append(b, byte(0)) // OID STRING LENGTH is 0
	return b
}

func (t *TX) UseCache() bool {
	return t.useCacheFrame
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
