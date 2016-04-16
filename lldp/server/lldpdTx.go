package lldpServer

import (
	_ "errors"
	_ "fmt"
	_ "github.com/google/gopacket"
	_ "github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
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

	}

	// we need to construct new lldp frame based of the information that we
	// have collected locally
}
