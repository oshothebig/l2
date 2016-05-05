package server

import (
	_ "encoding/binary"
	"errors"
	"fmt"
	_ "github.com/google/gopacket"
	_ "github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"l2/lldp/utils"
	_ "net"
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
		gblInfo.TxInfo.TxTimer =
			time.NewTimer(time.Duration(gblInfo.TxInfo.MessageTxInterval) *
				time.Second)
		svr.lldpGblInfo[ifIndex] = gblInfo
		// Start lldpMessage Tx interval
		<-gblInfo.TxInfo.TxTimer.C

		svr.lldpTxPktCh <- SendPktChannel{
			ifIndex: ifIndex,
		}
	}
}

/*  Write packet is helper function to send packet on wire.
 *  It will inform caller that packet was send successfully and you can go ahead
 *  and cache the pkt or else do not cache the packet as it is corrupted or there
 *  was some error
 */
func (gblInfo *LLDPGlobalInfo) WritePacket(pkt []byte) bool {
	var err error
	if len(pkt) == 0 {
		return false
	}
	gblInfo.PcapHdlLock.Lock()
	if gblInfo.PcapHandle != nil {
		err = gblInfo.PcapHandle.WritePacketData(pkt)
	} else {
		err = errors.New("Pcap Handle is invalid for " + gblInfo.Name)
	}
	gblInfo.PcapHdlLock.Unlock()
	if err != nil {
		debug.Logger.Err(fmt.Sprintln("Sending packet failed Error:",
			err, "for Port:", gblInfo.Name))
		return false
	}
	return true
}
